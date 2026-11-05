package application

import (
	"encoding/json"
	"github.com/dujiao-next/internal/constants"
	procurementdomain "github.com/dujiao-next/internal/modules/procurement/domain"
	"testing"
	"time"

	"github.com/dujiao-next/internal/shared/jsonmap"
)

func TestFrozenPlanProcurementPreservesExactEntitlementAndRoute(t *testing.T) {
	raw := `{"kind":"plan","issue_api_key":true,"plan":{"quota":9007199254740993}}`
	snapshot := jsonmap.JSON{"plan_checkout_entitlement_raw": raw, "plan_checkout_connection_id": "77", "plan_checkout_upstream_sku_id": "901"}
	contract, connectionID, skuID, present, err := procurementdomain.FrozenPlanProcurement(snapshot)
	if err != nil || !present || connectionID != 77 || skuID != 901 {
		t.Fatalf("frozen route invalid: %d %d %t %v", connectionID, skuID, present, err)
	}
	encoded, _ := json.Marshal(contract)
	if string(encoded) != `{"issue_api_key":true,"kind":"plan","plan":{"quota":9007199254740993}}` {
		t.Fatalf("quota lost precision: %s", encoded)
	}
	snapshot["plan_checkout_upstream_sku_id"] = "0"
	if _, _, _, _, err := procurementdomain.FrozenPlanProcurement(snapshot); err == nil {
		t.Fatal("malformed frozen route accepted")
	}
	if _, _, _, present, err := procurementdomain.FrozenPlanProcurement(jsonmap.JSON{}); err != nil || present {
		t.Fatal("legacy paid snapshot behavior changed")
	}
}

// ── Unit tests for pure functions ──

func TestParseRetryIntervals(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []time.Duration
	}{
		{
			name:     "empty string returns defaults",
			input:    "",
			expected: []time.Duration{30 * time.Second, 60 * time.Second, 300 * time.Second},
		},
		{
			name:     "valid array",
			input:    "[10,20,30]",
			expected: []time.Duration{10 * time.Second, 20 * time.Second, 30 * time.Second},
		},
		{
			name:     "with spaces",
			input:    "[ 10 , 20 , 30 ]",
			expected: []time.Duration{10 * time.Second, 20 * time.Second, 30 * time.Second},
		},
		{
			name:     "invalid entries skipped",
			input:    "[10,abc,30]",
			expected: []time.Duration{10 * time.Second, 30 * time.Second},
		},
		{
			name:     "all invalid returns defaults",
			input:    "[abc,def]",
			expected: []time.Duration{30 * time.Second, 60 * time.Second, 300 * time.Second},
		},
		{
			name:     "negative values skipped",
			input:    "[10,-5,30]",
			expected: []time.Duration{10 * time.Second, 30 * time.Second},
		},
		{
			name:     "zero values skipped",
			input:    "[0,10]",
			expected: []time.Duration{10 * time.Second},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRetryIntervals(tt.input)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d intervals, got %d: %v", len(tt.expected), len(result), result)
			}
			for i, d := range result {
				if d != tt.expected[i] {
					t.Errorf("interval[%d]: expected %v, got %v", i, tt.expected[i], d)
				}
			}
		})
	}
}

func TestBuildUpstreamRefundRecords_SortsByCreatedAtAscAndRenumbersID(t *testing.T) {
	records := []jsonmap.JSON{
		{
			"id":         99,
			"type":       "wallet",
			"amount":     "20.00",
			"created_at": "2026-04-12T10:00:00Z",
		},
		{
			"id":         100,
			"type":       "wallet",
			"amount":     "10.00",
			"created_at": "2026-04-12T09:00:00Z",
		},
	}

	got := buildUpstreamRefundRecords(records)
	if len(got) != 2 {
		t.Fatalf("expected 2 records, got %d", len(got))
	}
	if amount, _ := got[0]["amount"].(string); amount != "10.00" {
		t.Fatalf("expected first amount 10.00, got %#v", got[0]["amount"])
	}
	if amount, _ := got[1]["amount"].(string); amount != "20.00" {
		t.Fatalf("expected second amount 20.00, got %#v", got[1]["amount"])
	}
	if id, ok := got[0]["id"].(int); !ok || id != 1 {
		t.Fatalf("expected first id 1, got %#v", got[0]["id"])
	}
	if id, ok := got[1]["id"].(int); !ok || id != 2 {
		t.Fatalf("expected second id 2, got %#v", got[1]["id"])
	}
}

func TestIsRetryableErrorCode(t *testing.T) {
	nonRetryable := []string{
		"insufficient_balance",
		"payment_failed",
		"product_unavailable",
		"sku_unavailable",
		"invalid_request",
		"unauthorized",
		"forbidden",
		"duplicate_order",
		"product_out_of_stock",
	}
	for _, code := range nonRetryable {
		if isRetryableErrorCode(code) {
			t.Errorf("expected %q to be non-retryable", code)
		}
	}

	retryable := []string{
		"server_error",
		"timeout",
		"network_error",
		"unknown_error",
		"",
	}
	for _, code := range retryable {
		if !isRetryableErrorCode(code) {
			t.Errorf("expected %q to be retryable", code)
		}
	}

	// 测试带空格的情况
	if isRetryableErrorCode("  unauthorized  ") {
		t.Error("expected trimmed 'unauthorized' to be non-retryable")
	}
}

func TestIsUpstreamTransitionAllowed(t *testing.T) {
	for _, tc := range []struct {
		current, upstream string
		want              bool
	}{
		{constants.ProcurementStatusAccepted, "delivered", true},
		{constants.ProcurementStatusPartiallyRefunded, "delivered", true},
		{constants.ProcurementStatusFulfilled, "delivered", false},
		{constants.ProcurementStatusRefunded, "delivered", false},
		{constants.ProcurementStatusCanceled, "delivered", false},
		{constants.ProcurementStatusFulfilled, "canceled", false},
		{constants.ProcurementStatusAccepted, "canceled", true},
		{constants.ProcurementStatusAccepted, "partially_refunded", true},
		{constants.ProcurementStatusFulfilled, "refunded", true},
	} {
		if got := isUpstreamTransitionAllowed(tc.current, tc.upstream); got != tc.want {
			t.Errorf("(%s -> %s) = %v, want %v", tc.current, tc.upstream, got, tc.want)
		}
	}
}
