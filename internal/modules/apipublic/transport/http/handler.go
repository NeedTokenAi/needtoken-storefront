package apipublichttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	reseller "github.com/dujiao-next/internal/modules/reseller/contract"
	"github.com/gin-gonic/gin"
)

const (
	defaultPricingTimeout = 6 * time.Second
	defaultPricingTTL     = 60 * time.Second

	defaultPoolStatusTimeout = 5 * time.Second
	defaultPoolStatusTTL     = 30 * time.Second

	defaultStaleWindow = 10 * time.Minute
	maxUpstreamBody    = 4 << 20
)

var (
	emptyPricingJSON    = []byte(`{"success":true,"data":[],"vendors":[],"group_ratio":{},"usable_group":{}}`)
	emptyPoolStatusJSON = []byte(`{"groups":[]}`)
	unavailableJSON     = []byte(`{"success":false,"error_code":"upstream_unavailable"}`)
)

// BaseURLResolver resolves the active Glue upstream base URL for the main storefront.
type BaseURLResolver interface {
	ResolveGlueBaseURL(ctx context.Context) (baseURL string, configured bool, err error)
}

// Options configures optional overrides for tests.
type Options struct {
	HTTPClient        *http.Client
	Now               func() time.Time
	PricingTimeout    time.Duration
	PricingTTL        time.Duration
	PoolStatusTimeout time.Duration
	PoolStatusTTL     time.Duration
	StaleWindow       time.Duration
}

type cacheSlot struct {
	mu        sync.Mutex
	fetchMu   sync.Mutex
	baseURL   string
	body      []byte
	fetchedAt time.Time
}

// Handler serves same-origin cached public API pricing and pool status snapshots.
type Handler struct {
	resolver          BaseURLResolver
	client            *http.Client
	now               func() time.Time
	pricingTimeout    time.Duration
	pricingTTL        time.Duration
	poolStatusTimeout time.Duration
	poolStatusTTL     time.Duration
	staleWindow       time.Duration
	pricingCache      cacheSlot
	poolStatusCache   cacheSlot
}

func NewHandler(resolver BaseURLResolver, opts ...Options) *Handler {
	var opt Options
	if len(opts) > 0 {
		opt = opts[0]
	}
	client := opt.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	nowFn := opt.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	pricingTimeout := opt.PricingTimeout
	if pricingTimeout <= 0 {
		pricingTimeout = defaultPricingTimeout
	}
	pricingTTL := opt.PricingTTL
	if pricingTTL <= 0 {
		pricingTTL = defaultPricingTTL
	}
	poolTimeout := opt.PoolStatusTimeout
	if poolTimeout <= 0 {
		poolTimeout = defaultPoolStatusTimeout
	}
	poolTTL := opt.PoolStatusTTL
	if poolTTL <= 0 {
		poolTTL = defaultPoolStatusTTL
	}
	staleWindow := opt.StaleWindow
	if staleWindow <= 0 {
		staleWindow = defaultStaleWindow
	}
	return &Handler{
		resolver:          resolver,
		client:            client,
		now:               nowFn,
		pricingTimeout:    pricingTimeout,
		pricingTTL:        pricingTTL,
		poolStatusTimeout: poolTimeout,
		poolStatusTTL:     poolTTL,
		staleWindow:       staleWindow,
	}
}

func RegisterPublicRoutes(group gin.IRoutes, handler *Handler) {
	group.GET("/api-pricing", handler.GetAPIPricing)
	group.GET("/pool-status", handler.GetPoolStatus)
}

func (h *Handler) GetAPIPricing(c *gin.Context) {
	h.serveCachedUpstream(
		c,
		"/api/public/pricing",
		h.pricingTimeout,
		h.pricingTTL,
		emptyPricingJSON,
		&h.pricingCache,
	)
}

func (h *Handler) GetPoolStatus(c *gin.Context) {
	h.serveCachedUpstream(
		c,
		"/api/public/pool-status",
		h.poolStatusTimeout,
		h.poolStatusTTL,
		emptyPoolStatusJSON,
		&h.poolStatusCache,
	)
}

func (h *Handler) serveCachedUpstream(
	c *gin.Context,
	upstreamPath string,
	timeout time.Duration,
	freshTTL time.Duration,
	emptyBody []byte,
	slot *cacheSlot,
) {
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")

	tenant := reseller.TenantOrMain(c.Request.Context())
	if tenant.Unavailable || tenant.ResellerID != nil || h.resolver == nil {
		c.Data(http.StatusOK, "application/json; charset=utf-8", emptyBody)
		return
	}

	baseURL, configured, err := h.resolver.ResolveGlueBaseURL(c.Request.Context())
	if err != nil {
		if stale, ok := h.readStale(slot, ""); ok {
			c.Header("X-Storefront-Stale", "1")
			c.Data(http.StatusOK, "application/json; charset=utf-8", stale)
			return
		}
		c.Data(http.StatusServiceUnavailable, "application/json; charset=utf-8", unavailableJSON)
		return
	}
	if !configured {
		c.Data(http.StatusOK, "application/json; charset=utf-8", emptyBody)
		return
	}

	normalizedBase, ok := sanitizeUpstreamOrigin(baseURL)
	if !ok {
		if stale, ok := h.readStale(slot, ""); ok {
			c.Header("X-Storefront-Stale", "1")
			c.Data(http.StatusOK, "application/json; charset=utf-8", stale)
			return
		}
		c.Data(http.StatusServiceUnavailable, "application/json; charset=utf-8", unavailableJSON)
		return
	}

	if fresh, ok := h.readFresh(slot, normalizedBase, freshTTL); ok {
		c.Data(http.StatusOK, "application/json; charset=utf-8", fresh)
		return
	}

	slot.fetchMu.Lock()
	defer slot.fetchMu.Unlock()

	// Re-check inside fetchMu so concurrent cache misses share one upstream call.
	if fresh, ok := h.readFresh(slot, normalizedBase, freshTTL); ok {
		c.Data(http.StatusOK, "application/json; charset=utf-8", fresh)
		return
	}

	body, fetchErr := h.fetchJSONObject(c.Request.Context(), normalizedBase+upstreamPath, timeout)
	if fetchErr == nil {
		now := h.now()
		slot.mu.Lock()
		slot.baseURL = normalizedBase
		slot.body = body
		slot.fetchedAt = now
		slot.mu.Unlock()
		c.Data(http.StatusOK, "application/json; charset=utf-8", body)
		return
	}

	if stale, ok := h.readStale(slot, normalizedBase); ok {
		c.Header("X-Storefront-Stale", "1")
		c.Data(http.StatusOK, "application/json; charset=utf-8", stale)
		return
	}
	c.Data(http.StatusServiceUnavailable, "application/json; charset=utf-8", unavailableJSON)
}

func (h *Handler) readFresh(slot *cacheSlot, baseURL string, ttl time.Duration) ([]byte, bool) {
	now := h.now()
	slot.mu.Lock()
	defer slot.mu.Unlock()
	if len(slot.body) == 0 || slot.baseURL != baseURL {
		return nil, false
	}
	age := now.Sub(slot.fetchedAt)
	if age < 0 || age >= ttl {
		return nil, false
	}
	return slot.body, true
}

func (h *Handler) readStale(slot *cacheSlot, baseURL string) ([]byte, bool) {
	now := h.now()
	slot.mu.Lock()
	defer slot.mu.Unlock()
	if len(slot.body) == 0 {
		return nil, false
	}
	if baseURL != "" && slot.baseURL != baseURL {
		return nil, false
	}
	age := now.Sub(slot.fetchedAt)
	if age < 0 || age > h.staleWindow {
		return nil, false
	}
	return slot.body, true
}

func (h *Handler) fetchJSONObject(parent context.Context, targetURL string, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	client := *h.client
	client.Jar = nil
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("upstream status not ok")
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxUpstreamBody+1))
	if err != nil || len(raw) > maxUpstreamBody {
		return nil, errors.New("upstream body invalid")
	}

	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' || !json.Valid(trimmed) {
		return nil, errors.New("upstream payload is not a JSON object")
	}
	return trimmed, nil
}

func sanitizeUpstreamOrigin(raw string) (string, bool) {
	trimmed := strings.TrimRight(strings.TrimSpace(raw), "/")
	if trimmed == "" {
		return "", false
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", false
	}
	if (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" || parsed.User != nil {
		return "", false
	}
	if parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", false
	}
	return trimmed, true
}
