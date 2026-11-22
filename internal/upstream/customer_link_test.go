package upstream

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	siteconnectiondomain "github.com/dujiao-next/internal/modules/siteconnection/domain"
)

func TestCustomerLinkBridgeSignsRequestsMapsErrorsAndBlocksRedirects(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	var seen int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen++
		raw, _ := io.ReadAll(r.Body)
		ts, err := ParseTimestamp(r.Header.Get(HeaderTimestamp))
		if err != nil || r.Header.Get(HeaderApiKey) != "key" || !Verify("secret", r.Method, r.URL.Path, r.Header.Get(HeaderSignature), ts, raw) {
			t.Errorf("invalid hmac on %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/v1/upstream/customer-links/status":
			if r.URL.Query().Get("merchant_customer_id") != "needtoken-storefront:user:42" {
				t.Errorf("unexpected merchant_customer_id: %s", r.URL.Query().Get("merchant_customer_id"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":             true,
				"enabled":        true,
				"linked":         false,
				"status":         "unlinked",
				"pending_orders": 2,
			})
		case "/api/v1/upstream/customer-links/challenges":
			var body map[string]string
			if json.Unmarshal(raw, &body) != nil || body["merchant_customer_id"] != "needtoken-storefront:user:42" || body["next_path"] != "/plans" {
				t.Errorf("unexpected start body: %s", raw)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":            true,
				"challenge_id":  "lnk_123",
				"authorize_url": "https://api.needtoken.ai/account-link?challenge=v1.a.b",
				"expires_at":    now.Add(10 * time.Minute),
			})
		case "/api/v1/upstream/customer-links/complete":
			var body map[string]string
			_ = json.Unmarshal(raw, &body)
			if body["approval"] == "conflict" {
				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error_code": "native_account_already_linked"})
				return
			}
			if body["approval"] == "expired" {
				w.WriteHeader(http.StatusGone)
				_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error_code": "link_challenge_expired"})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":              true,
				"linked":          true,
				"status":          "self_verified",
				"native_user_id":  88,
				"next_path":       "/plans",
				"awakened_orders": 2,
			})
		case "/api/v1/upstream/customer-links/console-sso":
			var body CustomerConsoleSSOInput
			if json.Unmarshal(raw, &body) != nil || body.MerchantCustomerID != "needtoken-storefront:user:42" || body.Email != "alice@example.com" || !body.EmailVerified || body.TargetPath != "/keys" {
				t.Errorf("unexpected console-sso body: %s", raw)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"mode":            "sso",
				"redirect_url":    "https://api.needtoken.ai/oauth?redirect=%2Fkeys#needtoken_ticket=tok_1",
				"linked":          true,
				"native_user_id":  108,
				"auto_created":    true,
				"awakened_orders": 1,
			})
		case "/api/v1/upstream/customer-links/auto-link":
			var body CustomerAutoLinkInput
			if json.Unmarshal(raw, &body) != nil || body.MerchantCustomerID != "needtoken-storefront:user:42" {
				t.Errorf("unexpected auto-link body: %s", raw)
			}
			if body.Email == "existing@example.com" {
				// Simulate an upstream accidentally returning native_user_id; adapter must strip it.
				_ = json.NewEncoder(w).Encode(map[string]any{
					"ok":             true,
					"linked":         false,
					"status":         "existing_requires_link",
					"native_user_id": 999,
				})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":              true,
				"linked":          true,
				"status":          "auto_provisioned",
				"native_user_id":  108,
				"awakened_orders": 2,
			})
		case "/api/v1/upstream/customer-links/api-summary":
			if r.URL.Query().Get("storefront_user_id") != "needtoken-storefront:user:42" {
				t.Errorf("unexpected storefront_user_id: %s", r.URL.Query().Get("storefront_user_id"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":             true,
				"linked":         true,
				"native_user_id": 108,
				"quota":          5000000,
				"used_quota":     1250000,
				"request_count":  42,
				"group":          "vip",
				"active_subscriptions": []map[string]any{
					{
						"id":                    7,
						"plan_id":               3,
						"plan_title":            "Pro Monthly",
						"status":                "active",
						"amount_total":          10000000,
						"amount_used":           2000000,
						"start_time":            1700000000,
						"end_time":              1702592000,
						"allow_wallet_overflow": true,
					},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	adapter := NewDujiaoNextAdapter(&siteconnectiondomain.Connection{
		BaseURL:   server.URL,
		ApiKey:    "key",
		ApiSecret: "secret",
	}, "")

	ctx := context.Background()
	st, err := adapter.CustomerLinkStatus(ctx, "needtoken-storefront:user:42")
	if err != nil || !st.Enabled || st.Linked || st.PendingOrders != 2 {
		t.Fatalf("status=%+v err=%v", st, err)
	}
	ch, err := adapter.StartCustomerLink(ctx, "needtoken-storefront:user:42", "/plans")
	if err != nil || ch.ChallengeID != "lnk_123" || ch.AuthorizeURL == "" {
		t.Fatalf("challenge=%+v err=%v", ch, err)
	}
	done, err := adapter.CompleteCustomerLink(ctx, "needtoken-storefront:user:42", "lnk_123", "valid")
	if err != nil || !done.Linked || done.NativeUserID != 88 || done.AwakenedOrders != 2 {
		t.Fatalf("complete=%+v err=%v", done, err)
	}
	if _, err := adapter.CompleteCustomerLink(ctx, "needtoken-storefront:user:42", "lnk_123", "conflict"); !errors.Is(err, ErrCustomerLinkConflict) {
		t.Fatalf("want ErrCustomerLinkConflict, got %v", err)
	}
	if _, err := adapter.CompleteCustomerLink(ctx, "needtoken-storefront:user:42", "lnk_123", "expired"); !errors.Is(err, ErrCustomerLinkExpired) {
		t.Fatalf("want ErrCustomerLinkExpired, got %v", err)
	}
	ssoRes, err := adapter.ConsoleSSOCustomerLink(ctx, CustomerConsoleSSOInput{
		MerchantCustomerID: "needtoken-storefront:user:42",
		Email:              "alice@example.com",
		EmailVerified:      true,
		DisplayName:        "Alice",
		TargetPath:         "/keys",
		NextPath:           "/me",
	})
	if err != nil || ssoRes.Mode != "sso" || !ssoRes.Linked || ssoRes.NativeUserID != 108 || !ssoRes.AutoCreated {
		t.Fatalf("consoleSSO=%+v err=%v", ssoRes, err)
	}
	autoRes, err := adapter.AutoLink(ctx, CustomerAutoLinkInput{
		StorefrontUserID:   "needtoken-storefront:user:42",
		MerchantCustomerID: "needtoken-storefront:user:42",
		Email:              "alice@example.com",
		EmailVerified:      true,
		DisplayName:        "Alice",
	})
	if err != nil || !autoRes.Linked || autoRes.Status != "auto_provisioned" || autoRes.NativeUserID != 108 || autoRes.AwakenedOrders != 2 {
		t.Fatalf("autoLink=%+v err=%v", autoRes, err)
	}
	existingRes, err := adapter.AutoLink(ctx, CustomerAutoLinkInput{
		StorefrontUserID:   "needtoken-storefront:user:42",
		MerchantCustomerID: "needtoken-storefront:user:42",
		Email:              "existing@example.com",
		EmailVerified:      true,
	})
	if err != nil || existingRes.Linked || existingRes.Status != "existing_requires_link" || existingRes.NativeUserID != 0 {
		t.Fatalf("existing_requires_link must never leak native_user_id: %+v err=%v", existingRes, err)
	}
	summaryRes, err := adapter.APISummary(ctx, "needtoken-storefront:user:42")
	if err != nil || !summaryRes.Linked || summaryRes.NativeUserID != 108 || summaryRes.Quota != 5000000 || summaryRes.UsedQuota != 1250000 || summaryRes.Group != "vip" || len(summaryRes.ActiveSubscriptions) != 1 || summaryRes.ActiveSubscriptions[0].PlanTitle != "Pro Monthly" {
		t.Fatalf("apiSummary=%+v err=%v", summaryRes, err)
	}
	if seen != 9 {
		t.Fatalf("unexpected request count: %d", seen)
	}

	// Ensure redirects never leak credentials.
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, server.URL+"/api/v1/upstream/customer-links/status", http.StatusTemporaryRedirect)
	}))
	defer redirect.Close()
	redirectAdapter := NewDujiaoNextAdapter(&siteconnectiondomain.Connection{
		BaseURL:   redirect.URL,
		ApiKey:    "key",
		ApiSecret: "secret",
	}, "")
	if _, err := redirectAdapter.CustomerLinkStatus(ctx, "needtoken-storefront:user:42"); !errors.Is(err, ErrCustomerLinkUnavailable) {
		t.Fatalf("redirect should fail closed: %v", err)
	}
	if seen != 9 {
		t.Fatalf("redirect leaked request to upstream: seen=%d", seen)
	}
}
