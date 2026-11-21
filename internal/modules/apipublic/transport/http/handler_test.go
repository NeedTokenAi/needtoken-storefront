package apipublichttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type stubResolver struct {
	baseURL    string
	configured bool
	err        error
}

func (s *stubResolver) ResolveGlueBaseURL(context.Context) (string, bool, error) {
	return s.baseURL, s.configured, s.err
}

func TestUnconfiguredGlueReturnsEmptySafePayloads(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	public := router.Group("/api/v1/public")
	RegisterPublicRoutes(public, NewHandler(&stubResolver{configured: false}))

	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/api/v1/public/api-pricing", nil))
	if w1.Code != http.StatusOK {
		t.Fatalf("pricing status = %d, want 200", w1.Code)
	}
	var pricing map[string]any
	if err := json.Unmarshal(w1.Body.Bytes(), &pricing); err != nil {
		t.Fatalf("decode pricing: %v", err)
	}
	if pricing["success"] != true {
		t.Fatalf("unexpected pricing fallback: %v", pricing)
	}

	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/api/v1/public/pool-status", nil))
	if w2.Code != http.StatusOK {
		t.Fatalf("pool-status status = %d, want 200", w2.Code)
	}
	if got := w2.Body.String(); got != `{"groups":[]}` {
		t.Fatalf("pool-status fallback = %s, want {\"groups\":[]}", got)
	}
}

func TestCachingAndTenMinuteStaleFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var pricingHits atomic.Int32
	var poolHits atomic.Int32
	var failUpstream atomic.Bool

	upstreamSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if failUpstream.Load() {
			http.Error(w, "bad gateway", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/public/pricing":
			pricingHits.Add(1)
			_, _ = w.Write([]byte(`{"success":true,"data":[{"model_name":"gpt-5.4","quota_type":0,"model_ratio":1.25,"completion_ratio":4}],"vendors":[{"id":1,"name":"OpenAI"}],"group_ratio":{"default":1},"usable_group":{"default":"标准分组"}}`))
		case "/api/public/pool-status":
			poolHits.Add(1)
			_, _ = w.Write([]byte(`{"enabled":true,"stale":false,"groups":{"default":{"status":"available"}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstreamSrv.Close()

	now := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)
	resolver := &stubResolver{baseURL: upstreamSrv.URL, configured: true}
	h := NewHandler(resolver, Options{
		HTTPClient: upstreamSrv.Client(),
		Now:        func() time.Time { return now },
	})

	router := gin.New()
	RegisterPublicRoutes(router.Group("/api/v1/public"), h)

	// 1. Cold fetch populates in-memory cache.
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/public/api-pricing", nil))
	if rec.Code != http.StatusOK || rec.Header().Get("X-Storefront-Stale") != "" {
		t.Fatalf("first pricing: code=%d stale=%q", rec.Code, rec.Header().Get("X-Storefront-Stale"))
	}
	if pricingHits.Load() != 1 {
		t.Fatalf("pricing hits = %d, want 1", pricingHits.Load())
	}

	// 2. Within 60s TTL, second call hits memory cache without calling upstream.
	now = now.Add(30 * time.Second)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/public/api-pricing", nil))
	if rec.Code != http.StatusOK || pricingHits.Load() != 1 {
		t.Fatalf("cached pricing: code=%d hits=%d", rec.Code, pricingHits.Load())
	}

	// 3. Past 60s TTL (at +2m) while upstream fails -> serves stale snapshot with X-Storefront-Stale: 1.
	failUpstream.Store(true)
	now = now.Add(90 * time.Second)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/public/api-pricing", nil))
	if rec.Code != http.StatusOK || rec.Header().Get("X-Storefront-Stale") != "1" {
		t.Fatalf("stale pricing fallback: code=%d stale=%q", rec.Code, rec.Header().Get("X-Storefront-Stale"))
	}

	// 4. Past 10-minute stale window while upstream still fails -> returns 503.
	now = now.Add(10 * time.Minute)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/public/api-pricing", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expired stale pricing: code=%d, want 503", rec.Code)
	}

	// 5. Pool status caches for 30s and degrades on transient error.
	failUpstream.Store(false)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/public/pool-status", nil))
	if rec.Code != http.StatusOK || poolHits.Load() != 1 {
		t.Fatalf("pool-status first: code=%d hits=%d", rec.Code, poolHits.Load())
	}
	now = now.Add(15 * time.Second)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/public/pool-status", nil))
	if rec.Code != http.StatusOK || poolHits.Load() != 1 {
		t.Fatalf("pool-status cached: code=%d hits=%d", rec.Code, poolHits.Load())
	}
	failUpstream.Store(true)
	now = now.Add(20 * time.Second)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/public/pool-status", nil))
	if rec.Code != http.StatusOK || rec.Header().Get("X-Storefront-Stale") != "1" {
		t.Fatalf("pool-status stale: code=%d stale=%q", rec.Code, rec.Header().Get("X-Storefront-Stale"))
	}
}
