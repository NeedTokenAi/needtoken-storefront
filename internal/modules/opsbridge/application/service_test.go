package application_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/modules/opsbridge/application"
	ordercontract "github.com/dujiao-next/internal/modules/order/contract"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	paymentcontract "github.com/dujiao-next/internal/modules/payment/contract"
	paymentdomain "github.com/dujiao-next/internal/modules/payment/domain"
	procurementcontract "github.com/dujiao-next/internal/modules/procurement/contract"
	procurementdomain "github.com/dujiao-next/internal/modules/procurement/domain"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/dujiao-next/internal/shared/money"
	"github.com/shopspring/decimal"
)

type orders struct{ ordercontract.Store }

func (orders) ListAdmin(ordercontract.ListFilter) ([]orderdomain.Order, int64, error) {
	couponID := uint(12)
	return []orderdomain.Order{{
		ID:             1,
		UserID:         7,
		OrderNo:        "test-order",
		Currency:       "CNY",
		OriginalAmount: money.FromDecimal(decimal.NewFromInt(100)),
		DiscountAmount: money.FromDecimal(decimal.NewFromInt(15)),
		TotalAmount:    money.FromDecimal(decimal.NewFromInt(85)),
		CouponID:       &couponID,
		GuestEmail:     "SECRET",
		GuestPassword:  "SECRET",
		Items:          []orderdomain.OrderItem{{ManualFormSubmissionJSON: jsonmap.JSON{"token": "SECRET"}}},
	}}, 1, nil
}
func (orders) ListRefundRecordsByOrderIDs([]uint) ([]orderdomain.OrderRefundRecord, error) {
	return []orderdomain.OrderRefundRecord{{ID: 4, Type: constants.OrderRefundTypeManual, Remark: "SECRET"}}, nil
}

type payments struct{ paymentcontract.Store }

func (payments) ListByOrderID(uint) ([]paymentdomain.Payment, error) {
	return []paymentdomain.Payment{{
		ID:              2,
		ProviderType:    "official",
		ChannelType:     "wechat",
		PayURL:          "SECRET",
		ProviderPayload: jsonmap.JSON{"token": "SECRET"},
	}}, nil
}
func (payments) ListCheckoutReservations(uint) ([]paymentdomain.CheckoutReservation, error) {
	return []paymentdomain.CheckoutReservation{{RootOrderID: 1, OrderID: 1, OrderItemID: 9, HoldID: "hold_opaque_01", HoldVersion: "9007199254740993", OrderRef: "sf_order_opaque_01", State: "needs_attention", BookStatus: "needs_attention", AttentionCode: "paid_hold_or_asset_requires_manual_review", ReserveRequestJSON: jsonmap.JSON{"secret": "SECRET"}, ConfirmRequestJSON: jsonmap.JSON{"secret": "SECRET"}, UpstreamIdentity: "SECRET"}}, nil
}

type procurements struct{ procurementcontract.Repository }

func (procurements) GetByLocalOrderID(uint) (*procurementdomain.Order, error) {
	return &procurementdomain.Order{ID: 3, UpstreamPayload: "SECRET"}, nil
}
func TestOpsWhitelistNeverSerializesCredentialsOrDeliveryPayload(t *testing.T) {
	result, err := application.New(orders{}, payments{}, procurements{}).Orders(0, 1000)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"SECRET", "guest_email", "guest_password", "manual_form", "provider_payload", "pay_url", "upstream_payload", "reserve_request", "confirm_request", "upstream_identity"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("sensitive field leaked: %s", data)
		}
	}
	if len(result.Orders[0].Reservations) != 1 || result.Orders[0].Reservations[0].Version != "9007199254740993" || result.Orders[0].Reservations[0].Status != "needs_attention" {
		t.Fatal("reservation trace or exact version missing")
	}
	if result.PageSize != 100 || result.Orders[0].MerchantCustomerID != "needtoken-storefront:user:7" || result.Orders[0].EntitlementStatus != "unknown" || result.Orders[0].Refunds[0].SettlementStatus != "external_unverified" || time.Since(result.CollectedAt) > time.Minute {
		t.Fatalf("invalid facts: %+v", result)
	}
	if result.Orders[0].OriginalAmount != "100.00" || result.Orders[0].DiscountAmount != "15.00" || result.Orders[0].TotalAmount != "85.00" || result.Orders[0].CouponID == nil || *result.Orders[0].CouponID != 12 {
		t.Fatalf("coupon and discount breakdown missing: %+v", result.Orders[0])
	}
	if len(result.Orders[0].Payments) != 1 || result.Orders[0].Payments[0].ProviderType != "official" || result.Orders[0].Payments[0].ChannelType != "wechat" {
		t.Fatalf("payment channel metadata missing: %+v", result.Orders[0].Payments)
	}
}
