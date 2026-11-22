package accountlinkhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/dujiao-next/internal/modules/order/application/accountlink"
	reseller "github.com/dujiao-next/internal/modules/reseller/contract"
	"github.com/gin-gonic/gin"
)

type fakeBridge struct {
	gotRef           string
	gotSSO           accountlink.ConsoleSSOInput
	gotAuto          accountlink.AutoLinkInput
	gotLogoutRef     string
	logoutRevoked    bool
	consoleCalls     []string
	failFirstNotLink bool
	consoleCallCount int
	err              error
}

func (f *fakeBridge) GatewayConsoleRequest(_ context.Context, method, ref, subPath string, query url.Values, rawBody []byte) (json.RawMessage, error) {
	f.gotRef = ref
	f.consoleCallCount++
	if f.failFirstNotLink && f.consoleCallCount == 1 {
		return nil, accountlink.ErrNotLinked
	}
	if f.err != nil {
		return nil, f.err
	}
	entry := method + " " + subPath
	if len(query) > 0 {
		entry += "?" + query.Encode()
	}
	f.consoleCalls = append(f.consoleCalls, entry)
	return json.RawMessage(`{"items":[],"total":0}`), nil
}

func (f *fakeBridge) StorefrontLogout(_ context.Context, ref, _ string) (bool, error) {
	f.gotLogoutRef = ref
	if f.err != nil {
		return false, f.err
	}
	return f.logoutRevoked, nil
}

type fakeSessionRevoker struct {
	revokedIDs []uint
}

func (f *fakeSessionRevoker) RevokeUserSessions(_ context.Context, userID uint) error {
	f.revokedIDs = append(f.revokedIDs, userID)
	return nil
}

func (f *fakeBridge) Status(_ context.Context, ref string) (accountlink.Status, error) {
	f.gotRef = ref
	if f.err != nil {
		return accountlink.Status{}, f.err
	}
	return accountlink.Status{Enabled: true, Linked: false, Status: "unlinked", PendingOrders: 1}, nil
}
func (f *fakeBridge) Start(_ context.Context, ref, _ string) (accountlink.Challenge, error) {
	f.gotRef = ref
	if f.err != nil {
		return accountlink.Challenge{}, f.err
	}
	return accountlink.Challenge{ChallengeID: "lnk_abc", AuthorizeURL: "https://api.needtoken.ai/account-link?challenge=v1.a.b", ExpiresAt: time.Now().Add(time.Minute)}, nil
}
func (f *fakeBridge) Complete(_ context.Context, ref, _, _ string) (accountlink.CompleteResult, error) {
	f.gotRef = ref
	if f.err != nil {
		return accountlink.CompleteResult{}, f.err
	}
	return accountlink.CompleteResult{Linked: true, Status: "self_verified", NativeUserID: 55, NextPath: "/plans", AwakenedOrders: 1}, nil
}
func (f *fakeBridge) ConsoleSSO(_ context.Context, input accountlink.ConsoleSSOInput) (accountlink.ConsoleSSOResult, error) {
	f.gotRef = input.MerchantCustomerID
	f.gotSSO = input
	if f.err != nil {
		return accountlink.ConsoleSSOResult{}, f.err
	}
	return accountlink.ConsoleSSOResult{
		Mode:         "sso",
		RedirectURL:  "https://api.needtoken.ai/oauth?redirect=%2Fkeys#needtoken_ticket=tok_xyz",
		Linked:       true,
		NativeUserID: 108,
	}, nil
}
func (f *fakeBridge) AutoLink(_ context.Context, input accountlink.AutoLinkInput) (accountlink.AutoLinkResult, error) {
	f.gotRef = input.MerchantCustomerID
	f.gotAuto = input
	if f.err != nil {
		return accountlink.AutoLinkResult{}, f.err
	}
	return accountlink.AutoLinkResult{
		Linked:         true,
		Status:         "auto_provisioned",
		NativeUserID:   108,
		AwakenedOrders: 1,
	}, nil
}
func (f *fakeBridge) APISummary(_ context.Context, ref string) (accountlink.APISummaryResult, error) {
	f.gotRef = ref
	if f.err != nil {
		return accountlink.APISummaryResult{}, f.err
	}
	return accountlink.APISummaryResult{
		Linked:       true,
		NativeUserID: 108,
		Quota:        5000000,
		UsedQuota:    1000000,
		RequestCount: 12,
		Group:        "default",
	}, nil
}

type fakeIdentity struct{}

func (fakeIdentity) LookupMember(_ context.Context, userID uint) (accountlink.MemberIdentity, error) {
	return accountlink.MemberIdentity{
		UserID:        userID,
		Email:         "verified-user@example.com",
		EmailVerified: true,
		DisplayName:   "Verified User",
		Active:        true,
	}, nil
}

func TestAccountLinkMemberHTTPFlowAndSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	bridge := &fakeBridge{}
	r := gin.New()
	authGroup := r.Group("", func(c *gin.Context) {
		c.Set("user_id", uint(18))
		c.Next()
	})
	svc := accountlink.New(bridge).WithIdentityReader(fakeIdentity{})
	RegisterMemberRoutes(authGroup, New(svc))

	// GET status
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me/api-account-link", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatalf("status=%d headers=%v body=%s", w.Code, w.Header(), w.Body.String())
	}
	if bridge.gotRef != "needtoken-storefront:user:18" {
		t.Fatalf("unexpected ref: %s", bridge.gotRef)
	}

	// POST start
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/me/api-account-link/start", bytes.NewBufferString(`{"next_path":"/plans"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("start status=%d body=%s", w.Code, w.Body.String())
	}

	// POST start with unknown field rejected
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/me/api-account-link/start", bytes.NewBufferString(`{"next_path":"/plans","merchant_customer_id":"spoof"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	var envelope struct {
		StatusCode int `json:"status_code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &envelope)
	if envelope.StatusCode == 0 {
		t.Fatalf("unknown field must be rejected: %s", w.Body.String())
	}

	// POST console-sso uses server-side email and rejects spoofed email fields
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/me/api-console-sso", bytes.NewBufferString(`{"target_path":"/keys","next_path":"/me"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || bridge.gotSSO.Email != "verified-user@example.com" || bridge.gotSSO.TargetPath != "/keys" {
		t.Fatalf("console-sso status=%d sso=%+v body=%s", w.Code, bridge.gotSSO, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/me/api-console-sso", bytes.NewBufferString(`{"target_path":"/keys","email":"attacker@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	envelope.StatusCode = 0
	_ = json.Unmarshal(w.Body.Bytes(), &envelope)
	if envelope.StatusCode == 0 {
		t.Fatalf("spoofed email field must be rejected: %s", w.Body.String())
	}

	// POST complete
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/me/api-account-link/complete", bytes.NewBufferString(`{"challenge_id":"lnk_abc","approval":"v1.a.b"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("complete status=%d body=%s", w.Code, w.Body.String())
	}

	// Conflict maps cleanly
	bridge.err = accountlink.ErrConflict
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/me/api-account-link/complete", bytes.NewBufferString(`{"challenge_id":"lnk_abc","approval":"v1.a.b"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	envelope.StatusCode = 0
	_ = json.Unmarshal(w.Body.Bytes(), &envelope)
	if envelope.StatusCode == 0 {
		t.Fatalf("conflict should return non-zero business status_code: %s", w.Body.String())
	}

	// POST ensure succeeds with empty body, rejects spoofed fields, and degrades on reseller tenant.
	bridge.err = nil
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/me/api-account-link/ensure", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" || bridge.gotAuto.Email != "verified-user@example.com" {
		t.Fatalf("ensure status=%d auto=%+v body=%s", w.Code, bridge.gotAuto, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/me/api-account-link/ensure", bytes.NewBufferString(`{"email":"spoof@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	envelope.StatusCode = 0
	_ = json.Unmarshal(w.Body.Bytes(), &envelope)
	if envelope.StatusCode == 0 {
		t.Fatalf("ensure must reject unknown body fields: %s", w.Body.String())
	}

	// GET api-account-summary succeeds on main tenant, rejects query params, and returns unlinked on reseller tenant.
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/me/api-account-summary", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" || bridge.gotRef != "needtoken-storefront:user:18" {
		t.Fatalf("api-summary status=%d body=%s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/me/api-account-summary?user_id=99", nil)
	r.ServeHTTP(w, req)
	envelope.StatusCode = 0
	_ = json.Unmarshal(w.Body.Bytes(), &envelope)
	if envelope.StatusCode == 0 {
		t.Fatalf("api-summary must reject query parameters: %s", w.Body.String())
	}

	resellerCtx := reseller.WithTenantContext(context.Background(), reseller.ResellerTenantContext("shop.example.com", 5, 50, "shop.example.com"))
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/me/api-account-link/ensure", nil).WithContext(resellerCtx)
	r.ServeHTTP(w, req)
	var ensureReseller struct {
		StatusCode int                        `json:"status_code"`
		Data       accountlink.AutoLinkResult `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &ensureReseller)
	if ensureReseller.StatusCode != 0 || ensureReseller.Data.Linked || ensureReseller.Data.Status != "disabled" {
		t.Fatalf("reseller ensure must degrade gracefully: %+v", ensureReseller)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/me/api-account-summary", nil).WithContext(resellerCtx)
	r.ServeHTTP(w, req)
	var summaryReseller struct {
		StatusCode int                          `json:"status_code"`
		Data       accountlink.APISummaryResult `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &summaryReseller)
	if summaryReseller.StatusCode != 0 || summaryReseller.Data.Linked {
		t.Fatalf("reseller api-summary must degrade gracefully: %+v", summaryReseller)
	}
}

func TestMemberLogoutSLO(t *testing.T) {
	gin.SetMode(gin.TestMode)
	bridge := &fakeBridge{logoutRevoked: true}
	revoker := &fakeSessionRevoker{}
	svc := accountlink.New(bridge).WithIdentityReader(fakeIdentity{}).WithSessionRevoker(revoker)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint(42))
		c.Next()
	})
	RegisterMemberRoutes(r, New(svc))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/me/logout", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || len(revoker.revokedIDs) != 1 || revoker.revokedIDs[0] != 42 || bridge.gotLogoutRef != "needtoken-storefront:user:42" {
		t.Fatalf("logout SLO failed: code=%d revoked=%v logoutRef=%s body=%s", w.Code, revoker.revokedIDs, bridge.gotLogoutRef, w.Body.String())
	}
}

func TestGatewayConsoleMemberRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	bridge := &fakeBridge{failFirstNotLink: true}
	svc := accountlink.New(bridge).WithIdentityReader(fakeIdentity{})
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint(88))
		c.Next()
	})
	RegisterMemberRoutes(r, New(svc))

	// First call hits ErrNotLinked -> transparently invokes AutoLink/Ensure and retries once!
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/me/gateway/tokens", nil))
	if w1.Code != http.StatusOK || bridge.gotAuto.MerchantCustomerID != "needtoken-storefront:user:88" {
		t.Fatalf("expected transparent auto-link retry on /me/gateway/tokens, code=%d auto=%+v body=%s", w1.Code, bridge.gotAuto, w1.Body.String())
	}

	// Create, update, delete, logs, usage routes work cleanly.
	createReq := httptest.NewRequest(http.MethodPost, "/me/gateway/tokens", bytes.NewBufferString(`{"name":"CLI"}`))
	createReq.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, createReq)
	if w2.Code != http.StatusOK {
		t.Fatalf("create token failed: code=%d body=%s", w2.Code, w2.Body.String())
	}

	updReq := httptest.NewRequest(http.MethodPut, "/me/gateway/tokens/9", bytes.NewBufferString(`{"status":2}`))
	updReq.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, updReq)
	if w3.Code != http.StatusOK {
		t.Fatalf("update token failed: code=%d body=%s", w3.Code, w3.Body.String())
	}

	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, httptest.NewRequest(http.MethodDelete, "/me/gateway/tokens/9", nil))
	if w4.Code != http.StatusOK {
		t.Fatalf("delete token failed: code=%d body=%s", w4.Code, w4.Body.String())
	}

	w5 := httptest.NewRecorder()
	r.ServeHTTP(w5, httptest.NewRequest(http.MethodGet, "/me/gateway/logs?p=1&page_size=20&model_name=claude-sonnet-4-6&drop=1", nil))
	if w5.Code != http.StatusOK {
		t.Fatalf("logs failed: code=%d body=%s", w5.Code, w5.Body.String())
	}

	w6 := httptest.NewRecorder()
	r.ServeHTTP(w6, httptest.NewRequest(http.MethodGet, "/me/gateway/usage?days=7", nil))
	if w6.Code != http.StatusOK {
		t.Fatalf("usage failed: code=%d body=%s", w6.Code, w6.Body.String())
	}

	if len(bridge.consoleCalls) != 6 || bridge.consoleCalls[4] != "GET /logs?model_name=claude-sonnet-4-6&p=1&page_size=20" {
		t.Fatalf("unexpected console proxy calls: %v", bridge.consoleCalls)
	}
}
