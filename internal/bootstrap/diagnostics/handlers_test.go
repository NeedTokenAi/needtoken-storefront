package diagnosticsbootstrap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dujiao-next/internal/app/httpserver/middleware"
	diagnosticscontract "github.com/dujiao-next/internal/modules/diagnostics/contract"
	diagnosticstransport "github.com/dujiao-next/internal/modules/diagnostics/transport/http"

	"github.com/gin-gonic/gin"
)

type fixtureInspector struct{ calls int }

func (f *fixtureInspector) Inspect(context.Context, diagnosticscontract.Input) (diagnosticscontract.Result, error) {
	f.calls++
	return diagnosticscontract.Result{Accounts: []diagnosticscontract.Account{}, UsageStatus: "not_requested"}, nil
}

func TestPublicDiagnosticRateLimitRunsBeforeInspectorAndKeepsNoStore(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	inspector := &fixtureInspector{}
	group := router.Group("/api/v1/public/tools")
	group.Use(diagnosticstransport.NoStore(), middleware.RateLimitMiddleware(nil, middleware.RateLimitRule{Prefix: "diagnostic-test", WindowSeconds: 60, MaxRequests: 1}, middleware.KeyByIP))
	diagnosticstransport.RegisterPublicRoutes(group, diagnosticstransport.NewHandler(inspector))
	for i := 0; i < 2; i++ {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/public/tools/codex-account", strings.NewReader(`{"access_token":"fixture-access-token"}`))
		request.Header.Set("Content-Type", "application/json")
		request.RemoteAddr = "192.0.2.20:5000"
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if i == 1 && recorder.Code != http.StatusTooManyRequests {
			t.Fatalf("rate limit not enforced: %d", recorder.Code)
		}
		if !strings.Contains(recorder.Header().Get("Cache-Control"), "no-store") {
			t.Fatal("rate-limit response lacks no-store")
		}
		if strings.Contains(recorder.Body.String(), "fixture-access-token") {
			t.Fatal("credential leaked")
		}
	}
	if inspector.calls != 1 {
		t.Fatalf("limited request reached upstream: %d", inspector.calls)
	}
}

type statusReaderFixture struct{ calls int }

func (f *statusReaderFixture) Latest(context.Context, diagnosticscontract.ServiceStatusProvider) (diagnosticscontract.ServiceStatusSnapshot, error) {
	f.calls++
	return diagnosticscontract.ServiceStatusSnapshot{Provider: diagnosticscontract.ServiceStatusClaude, SourceURL: "https://status.claude.com/", FetchedAt: time.Now().UTC()}, nil
}

func TestPublicStatusReadLimitDoesNotConsumeAccountCredentialLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	inspector, statuses := &fixtureInspector{}, &statusReaderFixture{}
	account := router.Group("/api/v1/public/tools", diagnosticstransport.NoStore(), middleware.RateLimitMiddleware(nil, middleware.RateLimitRule{Prefix: "status-test-account", WindowSeconds: 60, MaxRequests: 1}, middleware.KeyByIP))
	read := router.Group("/api/v1/public/tools", diagnosticstransport.NoStore(), middleware.RateLimitMiddleware(nil, middleware.RateLimitRule{Prefix: "status-test-read", WindowSeconds: 60, MaxRequests: 2}, middleware.KeyByIP))
	diagnosticstransport.RegisterPublicRoutes(account, diagnosticstransport.NewHandler(inspector))
	diagnosticstransport.RegisterServiceStatusRoutes(read, diagnosticstransport.NewServiceStatusHandler(statuses))
	for i := 0; i < 3; i++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/v1/public/tools/service-status/claude", nil)
		request.RemoteAddr = "192.0.2.22:5000"
		router.ServeHTTP(recorder, request)
		if (i < 2 && recorder.Code != 200) || (i == 2 && recorder.Code != 429) || !strings.Contains(recorder.Header().Get("Cache-Control"), "no-store") {
			t.Fatal("status read limit or no-store behavior changed")
		}
	}
	if statuses.calls != 2 {
		t.Fatal("rate-limited status request reached reader")
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/public/tools/codex-account", strings.NewReader(`{"access_token":"fixture-access-token"}`))
	request.RemoteAddr = "192.0.2.22:5000"
	router.ServeHTTP(recorder, request)
	if recorder.Code != 200 || inspector.calls != 1 || !strings.Contains(recorder.Body.String(), `"status_code":0`) {
		t.Fatal("public status reads consumed account credential limit")
	}
}
