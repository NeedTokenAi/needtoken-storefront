package upstream

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	subscriptionauth "github.com/dujiao-next/internal/modules/order/application/subscriptionauthorization"
)

func TestSubscriptionBridgeSignsExactBodyAndNeverFollowsRedirect(t *testing.T) {
	var seen int
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen++
		raw, _ := io.ReadAll(r.Body)
		stamp, err := ParseTimestamp(r.Header.Get(HeaderTimestamp))
		if err != nil || r.Header.Get(HeaderApiKey) != "key" || !Verify("secret", r.Method, r.URL.Path, r.Header.Get(HeaderSignature), stamp, raw) {
			t.Error("request signature mismatch")
			w.WriteHeader(401)
			return
		}
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "order_id": uint(9), "order_no": "UP-9", "snapshot_hash": strings.Repeat("a", 64), "authorization_platform": "openai", "route": "web_card", "plan": "plus-monthly", "period": "month"})
			return
		}
		var a subscriptionauth.Assertion
		if json.Unmarshal(raw, &a) != nil || a.ExecutionScope != "inspection_only" {
			t.Error("inspection assertion changed")
		}
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "accepted": true, "operation_id": a.OperationID})
	}))
	defer server.Close()
	a := &DujiaoNextAdapter{baseURL: server.URL, apiKey: "key", apiSecret: "secret", client: server.Client()}
	ctx := context.Background()
	v, err := a.SubscriptionAuthorizationContext(ctx, 9)
	if err != nil || v.Route != "web_card" {
		t.Fatal(err)
	}
	if err = a.OpenSubscriptionAuthorization(ctx, subscriptionauth.Assertion{OperationID: "auth_test", ExecutionScope: "inspection_only"}); err != nil {
		t.Fatal(err)
	}
	if seen != 2 {
		t.Fatal("wrong requests")
	}
	path := "/api/v1/subscription-authorization-consents/auth_test"
	stamp := time.Now().Unix()
	signature := subscriptionauth.Signature{APIKey: "key", Path: path, Timestamp: stamp, Value: Sign("secret", http.MethodGet, path, stamp, nil)}
	if !a.VerifySubscriptionAuthorization(signature) {
		t.Fatal("valid receipt signature rejected")
	}
	signature.Path += "other"
	if a.VerifySubscriptionAuthorization(signature) {
		t.Fatal("signature not path bound")
	}
	redirect := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, server.URL+"/steal", http.StatusTemporaryRedirect)
	}))
	defer redirect.Close()
	a.baseURL = redirect.URL
	a.client = redirect.Client()
	if _, err = a.SubscriptionAuthorizationContext(ctx, 9); err != ErrSubscriptionAuthorizationUnavailable {
		t.Fatal("redirect accepted")
	}
	if seen != 2 {
		t.Fatal("redirect leaked bridge credentials")
	}
}
func TestSubscriptionBridgeRejectsInsecureOrNonOriginBase(t *testing.T) {
	for _, base := range []string{"http://example.test", "https://user:password@example.test", "https://example.test/nested", "https://example.test?x=1"} {
		a := DujiaoNextAdapter{baseURL: base}
		if _, err := a.SubscriptionCredentialURL(); err == nil {
			t.Errorf("accepted %s", base)
		}
	}
}
