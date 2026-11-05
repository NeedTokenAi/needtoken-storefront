package upstream

import (
	"context"
	"net/http"
	"time"
)

// CheckPlanCheckoutEligibility signs the exact body with the existing Dujiao
// credentials. Redirects must not receive these credentials or the customer ref.
func (a *DujiaoNextAdapter) CheckPlanCheckoutEligibility(ctx context.Context, request PlanCheckoutEligibilityRequest) (*PlanCheckoutEligibilityResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	adapter := *a
	client := *a.client
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	adapter.client = &client
	var result PlanCheckoutEligibilityResult
	if err := adapter.doRequest(ctx, http.MethodPost, "/api/v1/upstream/checkouts/eligibility", request, &result); err != nil {
		return nil, ErrPlanCheckoutUnavailable
	}
	now := time.Now()
	if !result.OK || result.SKUID != request.SKUID || result.EntitlementContractHash != request.EntitlementContractHash ||
		result.CollectedAt.IsZero() || result.CollectedAt.Before(now.Add(-15*time.Second)) || result.CollectedAt.After(now.Add(5*time.Second)) {
		return nil, ErrPlanCheckoutUnavailable
	}
	if result.Ready && result.ReasonCode != "ready" || !result.Ready && result.ReasonCode == "ready" {
		return nil, ErrPlanCheckoutUnavailable
	}
	return &result, nil
}

var _ PlanCheckoutAdapter = (*DujiaoNextAdapter)(nil)
