package middleware

import "testing"

func TestDedicatedOpsCredentialScope(t *testing.T) {
	for _, test := range []struct {
		kind, path string
		allowed    bool
	}{
		{"needtoken_ops", "/api/v1/channel/needtoken-ops/overview", true},
		{"needtoken_ops", "/api/v1/channel/needtoken-ops/orders", true},
		{"telegram", "/api/v1/channel/needtoken-ops/orders", false},
		{"needtoken_ops", "/api/v1/channel/wallet/recharge", false},
		{"telegram", "/api/v1/channel/wallet/recharge", true},
		{"", "/api/v1/channel/needtoken-ops/orders", false},
	} {
		if got := channelTypeAllowsPath(test.kind, test.path); got != test.allowed {
			t.Errorf("%s %s: %v", test.kind, test.path, got)
		}
	}
}
