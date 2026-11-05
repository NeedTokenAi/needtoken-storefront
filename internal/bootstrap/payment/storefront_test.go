package paymentbootstrap

import (
	"errors"
	"testing"
	"time"

	mappingcontract "github.com/dujiao-next/internal/modules/catalog/mapping/contract"
	ordercontract "github.com/dujiao-next/internal/modules/order/contract"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	"github.com/dujiao-next/internal/modules/payment/application/plancheckout"
	paymentcontract "github.com/dujiao-next/internal/modules/payment/contract"
	paymentdomain "github.com/dujiao-next/internal/modules/payment/domain"
	"github.com/dujiao-next/internal/shared/jsonmap"
)

type pendingOrderReader struct {
	ordercontract.Store
	root *orderdomain.Order
}

func (r pendingOrderReader) GetByID(uint) (*orderdomain.Order, error) { return r.root, nil }

type pendingPaymentStore struct {
	paymentcontract.Store
	calls int
}

func (s *pendingPaymentStore) GetLatestPendingByOrder(uint, time.Time) (*paymentdomain.Payment, error) {
	s.calls++
	return &paymentdomain.Payment{PayURL: "https://payments.example.test/previous-link"}, nil
}

type pendingPlanChecker struct {
	calls int
	err   error
}

func (c *pendingPlanChecker) CheckPlanCheckout(mappingcontract.PlanCheckoutRequest) error {
	c.calls++
	return c.err
}

func TestLatestPaymentDoesNotExposeLegacyOrUnavailableAPIPlanLink(t *testing.T) {
	for _, test := range []struct {
		name               string
		userID             uint
		snapshot           jsonmap.JSON
		checkerError, want error
	}{
		{"legacy-missing-route", 7, jsonmap.JSON{"api_plan": true}, nil, mappingcontract.ErrPlanChanged},
		{"guest-legacy-link", 0, jsonmap.JSON{"api_plan": true}, nil, mappingcontract.ErrPlanAccountLinkRequired},
		{"existing-link-account-disabled", 7, jsonmap.JSON{"api_plan": true, "entitlement_contract": jsonmap.JSON{"kind": "plan"}, "entitlement_contract_hash": "frozen", "plan_checkout_connection_id": "11", "plan_checkout_upstream_sku_id": "22"}, mappingcontract.ErrPlanAccountUnavailable, mappingcontract.ErrPlanAccountUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := &orderdomain.Order{ID: 1, UserID: test.userID, Items: []orderdomain.OrderItem{{ID: 3, OrderID: 1, ProductID: 4, SKUID: 5, Quantity: 1, SKUSnapshotJSON: test.snapshot}}}
			payments := &pendingPaymentStore{}
			checker := &pendingPlanChecker{err: test.checkerError}
			adapter := pendingLookupAdapter{payments: payments, plans: plancheckout.New(pendingOrderReader{root: root}, nil, nil, checker)}
			payment, err := adapter.GetLatestPendingByOrder(1, time.Now())
			if !errors.Is(err, test.want) || payment != nil || payments.calls != 0 {
				t.Fatalf("ineligible existing payment URL exposed: %+v, %v", payment, err)
			}
		})
	}
}

func TestLatestOrdinaryPaymentStillReturnsExistingLink(t *testing.T) {
	root := &orderdomain.Order{ID: 1, Items: []orderdomain.OrderItem{{ID: 3, OrderID: 1, ProductID: 4, SKUID: 5, Quantity: 1}}}
	payments := &pendingPaymentStore{}
	adapter := pendingLookupAdapter{payments: payments, plans: plancheckout.New(pendingOrderReader{root: root}, nil, nil, nil)}
	payment, err := adapter.GetLatestPendingByOrder(1, time.Now())
	if err != nil || payment == nil || payment.PayURL == "" || payments.calls != 1 {
		t.Fatal("ordinary existing payment URL changed", err)
	}
}
