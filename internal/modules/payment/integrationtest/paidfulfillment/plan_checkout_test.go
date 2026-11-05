package paidfulfillment_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mappingcontract "github.com/dujiao-next/internal/modules/catalog/mapping/contract"
	productdomain "github.com/dujiao-next/internal/modules/catalog/product/domain"
	productstore "github.com/dujiao-next/internal/modules/catalog/product/store/gormstore"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	orderstore "github.com/dujiao-next/internal/modules/order/infrastructure/gormstore"
	paymentapp "github.com/dujiao-next/internal/modules/payment/application"
	paymentdomain "github.com/dujiao-next/internal/modules/payment/domain"
	walletapp "github.com/dujiao-next/internal/modules/wallet/application"
	walletstore "github.com/dujiao-next/internal/modules/wallet/infrastructure/gormstore"
	"github.com/dujiao-next/internal/shared/jsonmap"
)

type planCheckFunc func(mappingcontract.PlanCheckoutRequest) error

func (f planCheckFunc) CheckPlanCheckout(request mappingcontract.PlanCheckoutRequest) error {
	return f(request)
}

func planPaymentService(t *testing.T, fixture *checkoutFixture, checker mappingcontract.PlanCheckoutChecker, scarce bool) *paymentapp.PaymentService {
	t.Helper()
	var item orderdomain.OrderItem
	if err := fixture.db.Where("order_id = ?", fixture.order.ID).First(&item).Error; err != nil {
		t.Fatal(err)
	}
	snapshot := item.SKUSnapshotJSON
	snapshot["api_plan"] = true
	snapshot["entitlement_contract"] = jsonmap.JSON{"kind": "plan", "quota": "9007199254740993"}
	snapshot["plan_checkout_entitlement_raw"] = `{"kind":"plan","quota":9007199254740993}`
	snapshot["entitlement_contract_hash"] = "frozen-plan-hash"
	snapshot["plan_checkout_connection_id"] = "17"
	snapshot["plan_checkout_upstream_sku_id"] = "29"
	snapshot["requires_reservation"] = scarce
	if !scarce {
		snapshot["reservation_contract"] = nil
	}
	if err := fixture.db.Model(&item).Update("sku_snapshot_json", snapshot).Error; err != nil {
		t.Fatal(err)
	}
	if err := fixture.db.Model(&productdomain.ProductSKU{}).Where("id = ?", item.SKUID).Update("requires_reservation", scarce).Error; err != nil {
		t.Fatal(err)
	}
	wallets := walletstore.New(fixture.db)
	return paymentapp.NewPaymentService(paymentapp.PaymentServiceOptions{
		OrderStore: orderstore.New(fixture.db, "outbox-test-credential-secret"), ProductSKURepo: productstore.NewSKUStore(fixture.db), PaymentStore: fixture.payments,
		WalletRepo: wallets, WalletService: walletapp.NewService(walletapp.Options{Repository: wallets, Transactions: wallets}), CheckoutReservations: fixture.gate, PlanEligibility: checker,
	})
}

func TestPlanEligibilityDenialPrecedesAllPaymentAndReservationWrites(t *testing.T) {
	for _, input := range []paymentapp.CreatePaymentInput{{UseBalance: true}, {ChannelID: 1}, {ChannelID: 1, UseBalance: true}} {
		fixture := checkout(t, 1)
		calls := 0
		service := planPaymentService(t, fixture, planCheckFunc(func(mappingcontract.PlanCheckoutRequest) error {
			calls++
			return mappingcontract.ErrPlanAccountUnavailable
		}), true)
		input.OrderID = fixture.order.ID
		var before int64
		fixture.db.Model(&paymentdomain.Payment{}).Count(&before)
		for range 2 {
			if _, err := service.CreatePayment(input); !errors.Is(err, mappingcontract.ErrPlanAccountUnavailable) {
				t.Fatal("payment bypassed authoritative eligibility", err)
			}
		}
		var after int64
		fixture.db.Model(&paymentdomain.Payment{}).Count(&after)
		if calls != 2 || before != after || fixture.balance(t) != "500.00" || len(fixture.rows(t)) != 0 || fixture.remote.posts != 0 {
			t.Fatal("denied/retried payment wrote money, a link or a scarce reservation")
		}
	}
}

func TestPlanEligibilityRunsOutsideSQLiteTransactionAndAllowsWalletPayment(t *testing.T) {
	fixture := checkout(t, 1)
	calls := 0
	service := planPaymentService(t, fixture, planCheckFunc(func(request mappingcontract.PlanCheckoutRequest) error {
		calls++
		// The test DB has one connection. This query times out if payment holds
		// that connection while waiting for an upstream eligibility response.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		var count int64
		if err := fixture.db.WithContext(ctx).Model(&orderdomain.Order{}).Count(&count).Error; err != nil {
			t.Fatal("plan eligibility ran under a database transaction", err)
		}
		if request.UserID != 1 || request.Route.ConnectionID != 17 || request.Route.UpstreamSKUID != 29 || request.EntitlementContractHash != "frozen-plan-hash" || string(request.EntitlementContract) != `{"kind":"plan","quota":9007199254740993}` {
			t.Fatalf("payment sent a mutable plan or wrong customer: %+v", request)
		}
		return nil
	}), false)
	result, err := service.CreatePayment(paymentapp.CreatePaymentInput{OrderID: fixture.order.ID, UseBalance: true})
	if err != nil || result == nil || !result.OrderPaid || calls != 1 || fixture.balance(t) != "400.00" || fixture.remote.posts != 0 || len(fixture.rows(t)) != 0 {
		t.Fatal("eligible non-scarce API plan could not pay from wallet", err)
	}
}

func TestPlanEligibilityCannotPaySnapshotChangedDuringPreflight(t *testing.T) {
	fixture := checkout(t, 1)
	service := planPaymentService(t, fixture, planCheckFunc(func(mappingcontract.PlanCheckoutRequest) error {
		var item orderdomain.OrderItem
		if err := fixture.db.Where("order_id = ?", fixture.order.ID).First(&item).Error; err != nil {
			t.Fatal(err)
		}
		item.SKUSnapshotJSON["plan_checkout_upstream_sku_id"] = "30"
		return fixture.db.Model(&item).Update("sku_snapshot_json", item.SKUSnapshotJSON).Error
	}), false)
	if _, err := service.CreatePayment(paymentapp.CreatePaymentInput{OrderID: fixture.order.ID, UseBalance: true}); !errors.Is(err, mappingcontract.ErrPlanChanged) {
		t.Fatal("a changed plan reused previous eligibility", err)
	}
	if fixture.balance(t) != "500.00" {
		t.Fatal("changed plan debited the wallet")
	}
}
