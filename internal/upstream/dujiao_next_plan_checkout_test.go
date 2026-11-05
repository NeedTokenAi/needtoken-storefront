package upstream

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	siteconnectiondomain "github.com/dujiao-next/internal/modules/siteconnection/domain"
)

func TestPlanCheckoutSignsExactFrozenTermsAndValidatesFreshEcho(t *testing.T) {
	hash := strings.Repeat("a", 64)
	request := PlanCheckoutEligibilityRequest{SKUID: 901, Quantity: 1, EntitlementContract: json.RawMessage(`{"kind":"plan","issue_api_key":true,"plan":{"quota":9007199254740993}}`), EntitlementContractHash: hash, CustomerReference: PlanCheckoutCustomerReference{MerchantCustomerID: "needtoken-storefront:user:19", Source: "needtoken-storefront", Verification: "merchant_session"}}
	for _, mode := range []string{"ready", "stale", "future", "wrong-sku", "wrong-hash", "contradictory", "http-error", "invalid-json"} {
		t.Run(mode, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				if r.URL.Path != "/api/v1/upstream/checkouts/eligibility" || r.Method != http.MethodPost || !strings.Contains(string(body), "9007199254740993") {
					t.Errorf("wrong signed wire request: %s %s %s", r.Method, r.URL.Path, body)
				}
				timestamp, _ := ParseTimestamp(r.Header.Get("Dujiao-Next-Timestamp"))
				if r.Header.Get("Dujiao-Next-Api-Key") != "key" || r.Header.Get("Dujiao-Next-Signature") != Sign("secret", r.Method, r.URL.Path, timestamp, body) {
					t.Error("missing or incorrect merchant HMAC")
				}
				if mode == "http-error" {
					w.WriteHeader(401)
					return
				}
				if mode == "invalid-json" {
					_, _ = w.Write([]byte(`{"ok":`))
					return
				}
				result := PlanCheckoutEligibilityResult{OK: true, Ready: true, ReasonCode: "ready", SKUID: request.SKUID, EntitlementContractHash: hash, CollectedAt: time.Now().UTC()}
				switch mode {
				case "stale":
					result.CollectedAt = time.Now().Add(-time.Minute)
				case "future":
					result.CollectedAt = time.Now().Add(time.Minute)
				case "wrong-sku":
					result.SKUID++
				case "wrong-hash":
					result.EntitlementContractHash = strings.Repeat("b", 64)
				case "contradictory":
					result.ReasonCode = "account_link_required"
				}
				_ = json.NewEncoder(w).Encode(result)
			}))
			defer server.Close()
			adapter := NewDujiaoNextAdapter(&siteconnectiondomain.Connection{BaseURL: server.URL, ApiKey: "key", ApiSecret: "secret"}, "")
			_, err := adapter.CheckPlanCheckoutEligibility(context.Background(), request)
			if (err == nil) != (mode == "ready") {
				t.Fatalf("mode=%s error=%v", mode, err)
			}
		})
	}
}

func TestPlanCheckoutRedirectAndCancellationFailClosed(t *testing.T) {
	calls := 0
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
	defer target.Close()
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer origin.Close()
	adapter := NewDujiaoNextAdapter(&siteconnectiondomain.Connection{BaseURL: origin.URL, ApiKey: "key", ApiSecret: "secret"}, "")
	if _, err := adapter.CheckPlanCheckoutEligibility(context.Background(), PlanCheckoutEligibilityRequest{}); !errors.Is(err, ErrPlanCheckoutUnavailable) || calls != 0 {
		t.Fatalf("redirect followed or accepted: calls=%d error=%v", calls, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := adapter.CheckPlanCheckoutEligibility(ctx, PlanCheckoutEligibilityRequest{}); !errors.Is(err, ErrPlanCheckoutUnavailable) {
		t.Fatalf("cancellation accepted: %v", err)
	}
}
