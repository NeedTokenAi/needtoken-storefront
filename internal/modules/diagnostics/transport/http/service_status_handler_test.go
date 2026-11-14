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

type statusReaderFunc func(context.Context, diagnosticscontract.ServiceStatusProvider) (diagnosticscontract.ServiceStatusSnapshot, error)

func (f statusReaderFunc) Latest(ctx context.Context, provider diagnosticscontract.ServiceStatusProvider) (diagnosticscontract.ServiceStatusSnapshot, error) {
	return f(ctx, provider)
}

func statusRouter(reader diagnosticscontract.ServiceStatusReader) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterServiceStatusRoutes(router.Group("/api/v1/public/tools", NoStore()), NewServiceStatusHandler(reader))
	return router
}

func TestStatusHandlerPreservesProviderStaleAndSuccessfulRetrievalContract(t *testing.T) {
	for _, stale := range []bool{false, true} {
		now := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)
		router := statusRouter(statusReaderFunc(func(_ context.Context, provider diagnosticscontract.ServiceStatusProvider) (diagnosticscontract.ServiceStatusSnapshot, error) {
			return diagnosticscontract.ServiceStatusSnapshot{Provider: provider, SourceURL: "https://status.openai.com/", FetchedAt: now, Stale: stale, Summary: diagnosticscontract.ServiceStatusSummary{Components: []diagnosticscontract.ServiceStatusComponent{{ID: "gpt-login", Name: "Login", Status: "major_outage", GroupID: "chatgpt"}}}}, nil
		}))
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/public/tools/service-status/openai", nil))
		var envelope struct {
			StatusCode int                                       `json:"status_code"`
			Data       diagnosticscontract.ServiceStatusSnapshot `json:"data"`
		}
		if json.Unmarshal(recorder.Body.Bytes(), &envelope) != nil || recorder.Code != 200 || envelope.StatusCode != 0 || envelope.Data.Provider != "openai" || envelope.Data.Stale != stale || !envelope.Data.FetchedAt.Equal(now) || envelope.Data.Summary.Components[0].GroupID != "chatgpt" {
			t.Fatalf("status proxy contract changed: %s", recorder.Body.String())
		}
		if !strings.Contains(recorder.Header().Get("Cache-Control"), "no-store") {
			t.Fatal("status snapshot can be cached without freshness checks")
		}
	}
}

func TestStatusHandlerRejectsOtherProvidersBeforeSourceAndLocalizesFailures(t *testing.T) {
	calls := 0
	router := statusRouter(statusReaderFunc(func(context.Context, diagnosticscontract.ServiceStatusProvider) (diagnosticscontract.ServiceStatusSnapshot, error) {
		calls++
		return diagnosticscontract.ServiceStatusSnapshot{}, errors.New("PRIVATE_UPSTREAM_BODY")
	}))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/public/tools/service-status/other?url=https://untrusted.invalid", nil))
	if calls != 0 || recorder.Code != 200 || !strings.Contains(recorder.Body.String(), `"status_code":400`) || !strings.Contains(recorder.Body.String(), "invalid_provider") {
		t.Fatalf("unsupported provider reached source: %s", recorder.Body.String())
	}
	messages := make(map[string]bool)
	for _, locale := range []string{"zh-CN", "zh-TW", "en-US"} {
		recorder = httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/v1/public/tools/service-status/claude", nil)
		request.Header.Set("X-Lang", locale)
		router.ServeHTTP(recorder, request)
		var envelope struct {
			StatusCode int    `json:"status_code"`
			Msg        string `json:"msg"`
		}
		if json.Unmarshal(recorder.Body.Bytes(), &envelope) != nil || recorder.Code != 200 || envelope.StatusCode != 503 || envelope.Msg == "" || strings.HasPrefix(envelope.Msg, "error.") || strings.Contains(recorder.Body.String(), "PRIVATE_") {
			t.Fatalf("unsafe status error: %s", recorder.Body.String())
		}
		messages[envelope.Msg] = true
	}
	if len(messages) != 3 {
		t.Fatal("status unavailable message is not localized in all languages")
	}
}
