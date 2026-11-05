package upstream

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var ErrPlanCheckoutUnavailable = errors.New("upstream plan checkout eligibility unavailable")

// Plan checkout is optional. Ordinary upstreams retain the existing Adapter
// interface and cannot implicitly claim member-account eligibility support.
type PlanCheckoutAdapter interface {
	CheckPlanCheckoutEligibility(context.Context, PlanCheckoutEligibilityRequest) (*PlanCheckoutEligibilityResult, error)
}

type PlanCheckoutCustomerReference struct {
	MerchantCustomerID string `json:"merchant_customer_id"`
	Source             string `json:"source"`
	Verification       string `json:"verification"`
}

type PlanCheckoutEligibilityRequest struct {
	SKUID                   uint                          `json:"sku_id"`
	Quantity                int                           `json:"quantity"`
	EntitlementContract     json.RawMessage               `json:"entitlement_contract"`
	EntitlementContractHash string                        `json:"entitlement_contract_hash"`
	CustomerReference       PlanCheckoutCustomerReference `json:"customer_reference"`
}

type PlanCheckoutEligibilityResult struct {
	OK                      bool      `json:"ok"`
	Ready                   bool      `json:"ready"`
	ReasonCode              string    `json:"reason_code"`
	SKUID                   uint      `json:"sku_id"`
	EntitlementContractHash string    `json:"entitlement_contract_hash"`
	CollectedAt             time.Time `json:"collected_at"`
}
