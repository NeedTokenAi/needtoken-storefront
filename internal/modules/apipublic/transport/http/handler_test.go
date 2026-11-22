package apipublichttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	reseller "github.com/dujiao-next/internal/modules/reseller/contract"
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

func TestRankingsValidationPerPeriodCacheSingleflightAndTenantIsolation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var hitsByPeriod sync.Map
	var failUpstream atomic.Bool
	slowGate := make(chan struct{})
	var gateOnce sync.Once
	var delayToday atomic.Bool

	upstreamSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/public/rankings" {
			http.NotFound(w, r)
			return
		}
		period := r.URL.Query().Get("period")
		if period == "today" && delayToday.Load() {
			<-slowGate
		}
		if failUpstream.Load() {
			http.Error(w, "upstream error", http.StatusBadGateway)
			return
		}
		val, _ := hitsByPeriod.LoadOrStore(period, &atomic.Int32{})
		val.(*atomic.Int32).Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"models":[{"rank":1,"model_name":"gpt-5.4","vendor":"OpenAI","total_tokens":120000,"share":0.5,"growth_pct":12.5}],"vendors":[],"top_movers":[],"top_droppers":[]}}`))
	}))
	defer upstreamSrv.Close()

	now := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)
	h := NewHandler(&stubResolver{baseURL: upstreamSrv.URL, configured: true}, Options{
		HTTPClient: upstreamSrv.Client(),
		Now:        func() time.Time { return now },
	})
	router := gin.New()
	RegisterPublicRoutes(router.Group("/api/v1/public"), h)

	// Invalid period rejected with 400.
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/public/api-rankings?period=decade", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid period status = %d, want 400", rec.Code)
	}

	// Missing period defaults to "week", cached for 3 minutes.
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/public/api-rankings", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("default rankings status = %d, want 200", rec.Code)
	}
	now = now.Add(2 * time.Minute)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/public/api-rankings?period=week", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("cached week rankings status = %d", rec.Code)
	}
	weekHits, _ := hitsByPeriod.Load("week")
	if weekHits.(*atomic.Int32).Load() != 1 {
		t.Fatalf("week hits = %d, want 1", weekHits.(*atomic.Int32).Load())
	}

	// Independent cache slot for "month".
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/public/api-rankings?period=month", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("month rankings status = %d", rec.Code)
	}
	monthHits, _ := hitsByPeriod.Load("month")
	if monthHits.(*atomic.Int32).Load() != 1 {
		t.Fatalf("month hits = %d, want 1", monthHits.(*atomic.Int32).Load())
	}

	// Past 3m TTL while upstream fails -> serves 10m stale snapshot with X-Storefront-Stale: 1.
	failUpstream.Store(true)
	now = now.Add(2 * time.Minute) // total +4m since week fetch
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/public/api-rankings?period=week", nil))
	if rec.Code != http.StatusOK || rec.Header().Get("X-Storefront-Stale") != "1" {
		t.Fatalf("stale rankings fallback: code=%d stale=%q", rec.Code, rec.Header().Get("X-Storefront-Stale"))
	}

	// Past 10m stale window while upstream fails -> returns 503.
	now = now.Add(8 * time.Minute) // total +12m
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/public/api-rankings?period=week", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expired stale rankings status = %d, want 503", rec.Code)
	}

	// Singleflight deduplication on concurrent cold misses for "today".
	failUpstream.Store(false)
	delayToday.Store(true)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/public/api-rankings?period=today", nil))
			if w.Code != http.StatusOK {
				t.Errorf("concurrent today rankings status = %d", w.Code)
			}
		}()
	}
	time.Sleep(25 * time.Millisecond)
	gateOnce.Do(func() { close(slowGate) })
	wg.Wait()
	todayHits, _ := hitsByPeriod.Load("today")
	if todayHits.(*atomic.Int32).Load() != 1 {
		t.Fatalf("singleflight today hits = %d, want 1", todayHits.(*atomic.Int32).Load())
	}

	// Non-main (reseller) tenant returns empty safe snapshot without hitting upstream.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/api-rankings?period=year", nil)
	resellerCtx := reseller.WithTenantContext(req.Context(), reseller.ResellerTenantContext("shop.example.com", 9, 99, "shop.example.com"))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req.WithContext(resellerCtx))
	if rec.Code != http.StatusOK {
		t.Fatalf("reseller rankings status = %d, want 200", rec.Code)
	}
	if _, loaded := hitsByPeriod.Load("year"); loaded {
		t.Fatalf("reseller tenant must not hit Glue rankings upstream")
	}
}
