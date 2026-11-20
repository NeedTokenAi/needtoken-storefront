package upstream

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	siteconnectiondomain "github.com/dujiao-next/internal/modules/siteconnection/domain"
)

func TestAssetClaimBridgeSignsRequestsMapsExhaustedAndBlocksRedirects(t *testing.T) {
	var revealed int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		ts, err := ParseTimestamp(r.Header.Get(HeaderTimestamp))
		if err != nil || r.Header.Get(HeaderApiKey) != "key" || !Verify("secret", r.Method, r.URL.Path, r.Header.Get(HeaderSignature), ts, raw) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/v1/upstream/orders/99/asset-claim":
			if r.URL.Query().Get("buyer_ref") != "needtoken-storefront:user:5" {
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error_code": "asset_claim_forbidden"})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":                true,
				"eligible":          true,
				"asset_id":          "av_1",
				"masked_identifier": "te***1@example.com",
				"max_reveals":       1,
				"reveal_count":      revealed,
				"remaining_reveals": 1 - revealed,
			})
		case "/api/v1/upstream/orders/99/asset-claim/reveal":
			if revealed >= 1 {
				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error_code": "asset_claim_exhausted"})
				return
			}
			revealed++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":                 true,
				"eligible":           true,
				"asset_id":           "av_1",
				"account_identifier": "test1@example.com",
				"secret_payload":     "SecretPass!123",
				"max_reveals":        1,
				"reveal_count":       1,
				"remaining_reveals":  0,
			})
		}
	}))
	defer server.Close()

	adapter := NewDujiaoNextAdapter(&siteconnectiondomain.Connection{BaseURL: server.URL, ApiKey: "key", ApiSecret: "secret"}, "")
	ctx := context.Background()

	if _, err := adapter.AssetClaimStatus(ctx, 99, "needtoken-storefront:user:9"); !errors.Is(err, ErrAssetClaimForbidden) {
		t.Fatalf("wrong buyer should fail forbidden, got %v", err)
	}
	st, err := adapter.AssetClaimStatus(ctx, 99, "needtoken-storefront:user:5")
	if err != nil || !st.Eligible || st.RemainingReveals != 1 {
		t.Fatalf("status=%+v err=%v", st, err)
	}
	rev, err := adapter.RevealAssetClaim(ctx, 99, "needtoken-storefront:user:5", "member_5")
	if err != nil || rev.AccountIdentifier != "test1@example.com" || rev.SecretPayload != "SecretPass!123" || rev.RemainingReveals != 0 {
		t.Fatalf("reveal=%+v err=%v", rev, err)
	}
	if _, err := adapter.RevealAssetClaim(ctx, 99, "needtoken-storefront:user:5", "member_5"); !errors.Is(err, ErrAssetClaimExhausted) {
		t.Fatalf("second reveal should return ErrAssetClaimExhausted, got %v", err)
	}

	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, server.URL+"/api/v1/upstream/orders/99/asset-claim", http.StatusTemporaryRedirect)
	}))
	defer redirect.Close()
	radapter := NewDujiaoNextAdapter(&siteconnectiondomain.Connection{BaseURL: redirect.URL, ApiKey: "key", ApiSecret: "secret"}, "")
	if _, err := radapter.AssetClaimStatus(ctx, 99, "needtoken-storefront:user:5"); !errors.Is(err, ErrAssetClaimUnavailable) {
		t.Fatalf("redirect must fail closed, got %v", err)
	}
}
