package upstream

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	siteconnectiondomain "github.com/dujiao-next/internal/modules/siteconnection/domain"
	"github.com/dujiao-next/internal/shared/jsonmap"
)

func catalogReservationContract() jsonmap.JSON {
	return jsonmap.JSON{"version": 1, "sku": "ChatGPT-Plus-1mo", "region": "US", "currency": "USD", "required_minor": "2500", "ttl_seconds": 900}
}

func TestSignedCatalogCarriesRequiredCapacityMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timestamp, err := ParseTimestamp(r.Header.Get(HeaderTimestamp))
		if err != nil || r.Header.Get(HeaderApiKey) != "catalog-key" || !Verify("catalog-secret", r.Method, r.URL.RequestURI(), r.Header.Get(HeaderSignature), timestamp, nil) {
			t.Error("catalog request was not signed")
			w.WriteHeader(401)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "product": UpstreamProduct{ID: 7, IsActive: true, RequiresReservation: true, ReservationContract: catalogReservationContract(), SKUs: []UpstreamSKU{{ID: 11, SKUCode: "US-Plus", IsActive: true}}}})
	}))
	defer server.Close()
	adapter := NewDujiaoNextAdapter(&siteconnectiondomain.Connection{BaseURL: server.URL, ApiKey: "catalog-key", ApiSecret: "catalog-secret"}, t.TempDir())
	product, err := adapter.GetProduct(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	required, contract, err := ReservationMetadata(product, product.SKUs[0])
	if err != nil || !required || contract["required_minor"] != "2500" {
		t.Fatal(required, contract, err)
	}
	product.ReservationContract["required_minor"] = "1"
	if contract["required_minor"] != "2500" {
		t.Fatal("metadata snapshot aliases catalog")
	}
}

func TestReservationMetadataNeverUsesManualFormAndCannotDowngradeParent(t *testing.T) {
	product := &UpstreamProduct{ManualFormSchema: jsonmap.JSON{"requires_reservation": true, "reservation_contract": catalogReservationContract()}}
	if required, contract, err := ReservationMetadata(product, UpstreamSKU{}); err != nil || required || contract != nil {
		t.Fatal("manual form became authoritative metadata", required, contract, err)
	}
	product.RequiresReservation = true
	product.ReservationContract = catalogReservationContract()
	sku := UpstreamSKU{RequiresReservation: false, ReservationContract: catalogReservationContract()}
	sku.ReservationContract["region"] = "DE"
	sku.ReservationContract["currency"] = "EUR"
	required, contract, err := ReservationMetadata(product, sku)
	if err != nil || !required || contract["region"] != "DE" {
		t.Fatal(required, contract, err)
	}
	sku.ReservationContract = nil
	required, contract, err = ReservationMetadata(product, sku)
	if err != nil || !required || contract["region"] != "US" {
		t.Fatal("false SKU flag downgraded parent requirement", required, contract, err)
	}
	product.RequiresReservation = false
	required, _, err = ReservationMetadata(product, sku)
	if err != nil || !required {
		t.Fatal("orphaned contract disabled reservation", err)
	}
}

func TestRequiredReservationMissingOrMalformedContractFailsClosed(t *testing.T) {
	for _, contract := range []jsonmap.JSON{nil, {}, {"version": 1}, {"version": 1, "sku": "ChatGPT-Plus-1mo", "region": "US", "currency": "USD", "required_minor": 2500, "ttl_seconds": 900}} {
		product := &UpstreamProduct{RequiresReservation: true, ReservationContract: contract}
		required, out, err := ReservationMetadata(product, UpstreamSKU{})
		if err == nil || !required || out != nil {
			t.Fatal("required malformed contract opened checkout", required, out, err)
		}
	}
	product := &UpstreamProduct{}
	required, contract, err := ReservationMetadata(product, UpstreamSKU{})
	if err != nil || required || !reflect.DeepEqual(contract, jsonmap.JSON(nil)) {
		t.Fatal("ordinary product behavior changed")
	}
}

func TestSignedCatalogDistinguishesAbsentPolicyFromExplicitFalse(t *testing.T) {
	for _, declared := range []bool{false, true} {
		t.Run(map[bool]string{false: "legacy_absent", true: "explicit_false"}[declared], func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				timestamp, err := ParseTimestamp(r.Header.Get(HeaderTimestamp))
				if err != nil || !Verify("catalog-secret", r.Method, r.URL.RequestURI(), r.Header.Get(HeaderSignature), timestamp, nil) {
					t.Error("unsigned catalog request")
					w.WriteHeader(401)
					return
				}
				sku := map[string]any{"id": 11, "sku_code": "US-Plus", "is_active": true}
				product := map[string]any{"id": 7, "is_active": true, "skus": []any{sku}, "reservation_policy_declared": true}
				if declared {
					product["requires_reservation"] = false
					sku["requires_reservation"] = false
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "product": product})
			}))
			defer server.Close()
			adapter := NewDujiaoNextAdapter(&siteconnectiondomain.Connection{BaseURL: server.URL, ApiKey: "catalog-key", ApiSecret: "catalog-secret"}, t.TempDir())
			product, err := adapter.GetProduct(context.Background(), 7)
			if err != nil {
				t.Fatal(err)
			}
			if product.ReservationPolicyDeclared != declared || product.SKUs[0].ReservationPolicyDeclared != declared || product.RequiresReservation || product.SKUs[0].RequiresReservation {
				t.Fatal("wire declaration presence was lost/invented", product)
			}
		})
	}
}

func TestReservationPolicyWireFieldMustBeBooleanAndCannotBeSpoofedByHiddenMarker(t *testing.T) {
	for _, raw := range []string{`{"requires_reservation":null}`, `{"requires_reservation":"false"}`, `{"requires_reservation":0}`, `{"skus":[{"requires_reservation":null}]}`} {
		var product UpstreamProduct
		if err := json.Unmarshal([]byte(raw), &product); err == nil {
			t.Fatal("accepted nonboolean policy field", raw)
		}
	}
	var product UpstreamProduct
	if err := json.Unmarshal([]byte(`{"reservation_policy_declared":true,"skus":[{"reservation_policy_declared":true}]}`), &product); err != nil {
		t.Fatal(err)
	}
	if product.ReservationPolicyDeclared || product.SKUs[0].ReservationPolicyDeclared {
		t.Fatal("JSON set an internal wire-presence marker")
	}
	if err := json.Unmarshal([]byte(`{"requires_reservation":false,"skus":[{"requires_reservation":false}]}`), &product); err != nil {
		t.Fatal(err)
	}
	if !product.ReservationPolicyDeclared || !product.SKUs[0].ReservationPolicyDeclared {
		t.Fatal("explicit false was not declared")
	}
	if err := json.Unmarshal([]byte(`{}`), &product); err != nil || product.ReservationPolicyDeclared {
		t.Fatal("reused decoder retained old declaration", err)
	}
}
