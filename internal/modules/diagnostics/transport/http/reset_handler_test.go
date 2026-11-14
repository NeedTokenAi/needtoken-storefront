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
