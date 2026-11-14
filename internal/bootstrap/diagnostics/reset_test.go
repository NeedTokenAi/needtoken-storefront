package diagnosticsbootstrap

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dujiao-next/internal/app/httpserver/middleware"
	diagnosticsapplication "github.com/dujiao-next/internal/modules/diagnostics/application"
	diagnosticscontract "github.com/dujiao-next/internal/modules/diagnostics/contract"
	"github.com/dujiao-next/internal/modules/diagnostics/infrastructure/codexresets"
	diagnosticstransport "github.com/dujiao-next/internal/modules/diagnostics/transport/http"

	"github.com/gin-gonic/gin"
)

type resetTransportFunc func(*http.Request) (*http.Response, error)

func (f resetTransportFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestResetRouteServesValidatedCachedPublicSource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var calls atomic.Int32
	client := codexresets.NewClient(&http.Client{Transport: resetTransportFunc(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		if req.URL.String() != "https://codex-resets.com/api/v1/resets?limit=100" || req.Header.Get("Authorization") != "" {
			t.Fatal("request contract changed")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Request: req, Body: io.NopCloser(strings.NewReader(`{"data":[{"id":"2120999993079451919","reset_type":"banked","announced_at":"2026-11-12T21:59:39.000Z","text":"Banked reset activated","source":{"type":"x_post","author":"thsottiaux","url":"https://x.com/thsottiaux/status/2120999993079451919"}}],"pagination":{"has_more":true},"meta":{"generated_at":"2099-01-01T00:00:00Z"}}`))}, nil
	})})
	cache := diagnosticsapplication.NewResetFeed(client)
	router := gin.New()
	diagnosticstransport.RegisterResetRoutes(router.Group("/api/v1/public/tools", diagnosticstransport.NoStore()), diagnosticstransport.NewResetHandler(cache))
	started := time.Now()
	var firstFetched time.Time
	for i := 0; i < 2; i++ {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/public/tools/codex-resets", nil))
		var envelope struct {
			StatusCode int                                   `json:"status_code"`
			Data       diagnosticscontract.ResetFeedSnapshot `json:"data"`
		}
		if json.Unmarshal(recorder.Body.Bytes(), &envelope) != nil || recorder.Code != 200 || envelope.StatusCode != 0 || len(envelope.Data.Records) != 1 || envelope.Data.Stale || !envelope.Data.HasMore || envelope.Data.Records[0].Type != "banked" || envelope.Data.SourceURL != "https://codex-resets.com/" {
			t.Fatalf("invalid handler/source contract: %s", recorder.Body.String())
		}
		if envelope.Data.FetchedAt.Before(started) || envelope.Data.FetchedAt.After(time.Now()) || strings.Contains(recorder.Body.String(), "2099-") || strings.Contains(recorder.Body.String(), "last_polled") {
			t.Fatal("fetched_at is not the local successful fetch time")
		}
		if i == 0 {
			firstFetched = envelope.Data.FetchedAt
		} else if !firstFetched.Equal(envelope.Data.FetchedAt) {
			t.Fatal("cache hit changed successful fetch timestamp")
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("handler bypassed cache: %d calls", calls.Load())
	}
}

type resetReaderFixture struct{}

func (resetReaderFixture) Latest(context.Context) (diagnosticscontract.ResetFeedSnapshot, error) {
	return diagnosticscontract.ResetFeedSnapshot{SourceURL: "https://codex-resets.com/", FetchedAt: time.Now().UTC(), Records: []diagnosticscontract.ResetRecord{}}, nil
}

func TestResetReadsDoNotConsumeAccountDiagnosticLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	inspector := &fixtureInspector{}
	account := router.Group("/api/v1/public/tools", diagnosticstransport.NoStore(), middleware.RateLimitMiddleware(nil, middleware.RateLimitRule{Prefix: "reset-test-account", WindowSeconds: 60, MaxRequests: 1}, middleware.KeyByIP))
	read := router.Group("/api/v1/public/tools", diagnosticstransport.NoStore(), middleware.RateLimitMiddleware(nil, middleware.RateLimitRule{Prefix: "reset-test-read", WindowSeconds: 60, MaxRequests: 30}, middleware.KeyByIP))
	diagnosticstransport.RegisterPublicRoutes(account, diagnosticstransport.NewHandler(inspector))
	diagnosticstransport.RegisterResetRoutes(read, diagnosticstransport.NewResetHandler(resetReaderFixture{}))
	for i := 0; i < 5; i++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/v1/public/tools/codex-resets", nil)
		request.RemoteAddr = "192.0.2.21:5000"
		router.ServeHTTP(recorder, request)
		if recorder.Code != 200 || !strings.Contains(recorder.Body.String(), `"status_code":0`) {
			t.Fatalf("unexpected read response: %s", recorder.Body.String())
		}
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/public/tools/codex-account", strings.NewReader(`{"access_token":"fixture-access-token"}`))
	request.RemoteAddr = "192.0.2.21:5000"
	router.ServeHTTP(recorder, request)
	if recorder.Code != 200 || inspector.calls != 1 || !strings.Contains(recorder.Body.String(), `"status_code":0`) {
		t.Fatalf("public feed consumed credential diagnostic limit: %s", recorder.Body.String())
	}
}
