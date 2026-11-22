package diagnosticshttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	diagnosticscontract "github.com/dujiao-next/internal/modules/diagnostics/contract"

	"github.com/gin-gonic/gin"
)

type resetReaderFunc func(context.Context) (diagnosticscontract.ResetFeedSnapshot, error)

func (f resetReaderFunc) Latest(ctx context.Context) (diagnosticscontract.ResetFeedSnapshot, error) {
	return f(ctx)
}

func resetRouter(reader diagnosticscontract.ResetFeedReader) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterResetRoutes(router.Group("/api/v1/public/tools", NoStore()), NewResetHandler(reader))
	return router
}

func TestResetHandlerReturnsAttributedSnapshotAndStaleContract(t *testing.T) {
	for _, stale := range []bool{false, true} {
		now := time.Date(2026, 11, 14, 1, 2, 3, 0, time.UTC)
		router := resetRouter(resetReaderFunc(func(context.Context) (diagnosticscontract.ResetFeedSnapshot, error) {
			return diagnosticscontract.ResetFeedSnapshot{SourceURL: "https://codex-resets.com/", FetchedAt: now, Stale: stale, HasMore: true, Records: []diagnosticscontract.ResetRecord{{ID: "123", Type: "regular", AnnouncedAt: now.Add(-time.Hour), Text: "Observed reset", Source: diagnosticscontract.ResetSource{Type: "observed"}}}}, nil
		}))
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/public/tools/codex-resets", nil))
		var envelope struct {
			StatusCode int                                   `json:"status_code"`
			Data       diagnosticscontract.ResetFeedSnapshot `json:"data"`
		}
		if json.Unmarshal(recorder.Body.Bytes(), &envelope) != nil || recorder.Code != 200 || envelope.StatusCode != 0 || envelope.Data.Stale != stale || envelope.Data.SourceURL != "https://codex-resets.com/" || !envelope.Data.FetchedAt.Equal(now) || !envelope.Data.HasMore || len(envelope.Data.Records) != 1 {
			t.Fatalf("wrong feed contract: %s", recorder.Body.String())
		}
		if !strings.Contains(recorder.Header().Get("Cache-Control"), "no-store") || strings.Contains(recorder.Body.String(), "generated_at") {
			t.Fatal("unsafe client cache or source timestamp relabeling")
		}
	}
}

func TestResetHandlerLocalizesUnavailableWithoutUpstreamDetails(t *testing.T) {
	var messages []string
	for _, locale := range []string{"zh-CN", "zh-TW", "en-US"} {
		router := resetRouter(resetReaderFunc(func(context.Context) (diagnosticscontract.ResetFeedSnapshot, error) {
			return diagnosticscontract.ResetFeedSnapshot{}, errors.New("PRIVATE_UPSTREAM_RESPONSE")
		}))
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/v1/public/tools/codex-resets", nil)
		request.Header.Set("X-Lang", locale)
		router.ServeHTTP(recorder, request)
		var envelope struct {
			StatusCode int    `json:"status_code"`
			Msg        string `json:"msg"`
			Data       struct {
				ErrorCode string `json:"error_code"`
			} `json:"data"`
		}
		if json.Unmarshal(recorder.Body.Bytes(), &envelope) != nil || recorder.Code != 200 || envelope.StatusCode != 503 || envelope.Data.ErrorCode != "unavailable" || envelope.Msg == "" || strings.HasPrefix(envelope.Msg, "error.") || strings.Contains(recorder.Body.String(), "PRIVATE_") {
			t.Fatalf("wrong unavailable contract: %s", recorder.Body.String())
		}
		messages = append(messages, envelope.Msg)
	}
	if messages[0] == messages[1] || messages[1] == messages[2] {
		t.Fatal("unavailable messages are not localized")
	}
}

type multiChannelReaderFixture struct {
	snapshots map[string]diagnosticscontract.ResetFeedSnapshot
}

func (f multiChannelReaderFixture) Latest(ctx context.Context) (diagnosticscontract.ResetFeedSnapshot, error) {
	return f.LatestForChannel(ctx, "codex")
}

func (f multiChannelReaderFixture) LatestForChannel(_ context.Context, channel string) (diagnosticscontract.ResetFeedSnapshot, error) {
	if snap, ok := f.snapshots[channel]; ok {
		return snap, nil
	}
	return diagnosticscontract.ResetFeedSnapshot{}, diagnosticscontract.ErrResetFeedUnavailable
}

func TestChannelResetHandlerSetsEdgeCacheHeadersAndRejectsUnknownChannel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 11, 22, 8, 0, 0, 0, time.UTC)
	router := gin.New()
	handler := NewResetHandler(multiChannelReaderFixture{
		snapshots: map[string]diagnosticscontract.ResetFeedSnapshot{
			"claude": {SourceURL: "https://www.tsalon.tech/whenreset/", FetchedAt: now, Records: []diagnosticscontract.ResetRecord{{ID: "c1", Type: "banked", AnnouncedAt: now, Text: "Claude reset card", TextZH: "Claude 重置卡"}}},
			"gemini": {SourceURL: "https://yidinghuiziyou.com/ai/antigravity-reset/", FetchedAt: now, Records: []diagnosticscontract.ResetRecord{{ID: "g1", Type: "regular", AnnouncedAt: now, Text: "Antigravity reset", TextZH: "Antigravity 全员重置"}}},
		},
	})
	RegisterChannelResetRoutes(router.Group("/api/v1/public/tools"), handler)

	for _, tc := range []struct {
		channel string
		wantCC  string
		wantID  string
	}{
		{"claude", "s-maxage=300", "c1"},
		{"gemini", "s-maxage=600", "g1"},
	} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/public/tools/resets/"+tc.channel, nil))
		if rec.Code != 200 || !strings.Contains(rec.Header().Get("Cache-Control"), tc.wantCC) || !strings.Contains(rec.Body.String(), tc.wantID) {
			t.Fatalf("channel %s unexpected response: code=%d cc=%q body=%s", tc.channel, rec.Code, rec.Header().Get("Cache-Control"), rec.Body.String())
		}
	}

	// Unknown channel rejected with 400 + no-store.
	bad := httptest.NewRecorder()
	router.ServeHTTP(bad, httptest.NewRequest(http.MethodGet, "/api/v1/public/tools/resets/unknown", nil))
	if bad.Code != 400 || !strings.Contains(bad.Header().Get("Cache-Control"), "no-store") {
		t.Fatalf("expected 400 no-store for unknown channel, got %d %q", bad.Code, bad.Header().Get("Cache-Control"))
	}

	// Unavailable channel never cached publicly.
	unavail := httptest.NewRecorder()
	router.ServeHTTP(unavail, httptest.NewRequest(http.MethodGet, "/api/v1/public/tools/resets/grok", nil))
	if !strings.Contains(unavail.Header().Get("Cache-Control"), "no-store") || !strings.Contains(unavail.Body.String(), `"status_code":503`) {
		t.Fatalf("expected 503 no-store for unavailable channel, got %d %q %s", unavail.Code, unavail.Header().Get("Cache-Control"), unavail.Body.String())
	}
}
