package accountlinkhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dujiao-next/internal/modules/order/application/accountlink"
	"github.com/gin-gonic/gin"
)

type fakeBridge struct {
	gotRef string
	gotSSO accountlink.ConsoleSSOInput
	err    error
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
}
