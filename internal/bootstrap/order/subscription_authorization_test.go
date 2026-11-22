package orderwiring

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/dujiao-next/internal/app/container"
	fulfillmentdomain "github.com/dujiao-next/internal/modules/fulfillment/domain"
	subscriptionauth "github.com/dujiao-next/internal/modules/order/application/subscriptionauthorization"
	ordercontract "github.com/dujiao-next/internal/modules/order/contract"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	orderstore "github.com/dujiao-next/internal/modules/order/infrastructure/gormstore"
	paymentcontract "github.com/dujiao-next/internal/modules/payment/contract"
	paymentdomain "github.com/dujiao-next/internal/modules/payment/domain"
	paymentstore "github.com/dujiao-next/internal/modules/payment/infrastructure/gormstore"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestAuthorizationDirectoryEnforcesRealOwnerGuestProofAndTenant(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:storefront_authorization_directory_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sql, _ := db.DB()
	sql.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sql.Close() })
	if err = db.AutoMigrate(&orderdomain.Order{}, &orderdomain.OrderItem{}, &fulfillmentdomain.Fulfillment{}); err != nil {
		t.Fatal(err)
	}
	store := orderstore.New(db, "test-order-secret")
	now := time.Now().UTC()
	resellerID := uint(7)
	orders := []orderdomain.Order{
		{ID: 1, OrderNo: "member", UserID: 1, GuestEmail: "same@example.test", Status: "paid", Currency: "USD", PaidAt: &now},
		{ID: 2, OrderNo: "guest", GuestEmail: "same@example.test", GuestPassword: "private-proof", Status: "paid", Currency: "USD", PaidAt: &now},
		{ID: 3, OrderNo: "reseller", UserID: 1, ResellerID: &resellerID, Status: "paid", Currency: "USD", PaidAt: &now},
	}
	for index := range orders {
		if err = store.Create(&orders[index], []orderdomain.OrderItem{{Quantity: 1, TitleJSON: jsonmap.JSON{"en-US": "Subscription"}}}); err != nil {
			t.Fatal(err)
		}
	}
	directory := authorizationDirectory{&container.Container{OrderStore: store}}
	cases := []struct {
		name    string
		subject subscriptionauth.Subject
		want    bool
	}{
		{"real member", subscriptionauth.Subject{OrderNo: "member", UserID: 1}, true},
		{"other member", subscriptionauth.Subject{OrderNo: "member", UserID: 2}, false},
		{"same email does not claim guest", subscriptionauth.Subject{OrderNo: "guest", UserID: 1}, false},
		{"guest private proof", subscriptionauth.Subject{OrderNo: "guest", GuestEmail: "same@example.test", GuestPassword: "private-proof"}, true},
		{"wrong private proof", subscriptionauth.Subject{OrderNo: "guest", GuestEmail: "same@example.test", GuestPassword: "wrong"}, false},
		{"cross tenant", subscriptionauth.Subject{OrderNo: "reseller", UserID: 1}, false},
		{"same tenant", subscriptionauth.Subject{OrderNo: "reseller", UserID: 1, ResellerID: &resellerID}, true},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			snapshot, err := directory.Resolve(context.Background(), test.subject)
			if test.want {
				if err != nil || !snapshot.Paid || snapshot.PaymentEnvironment != "unknown" {
					t.Fatalf("snapshot=%#v err=%v", snapshot, err)
				}
			} else if err != ordercontract.ErrAuthorizationNotFound {
				t.Fatalf("accepted identity: %#v %v", snapshot, err)
			}
		})
	}

	if err = db.AutoMigrate(&paymentdomain.Payment{}); err != nil {
		t.Fatal(err)
	}
	payments := paymentstore.New(db, "test-order-secret")
	paidAt := now
	if err = payments.Create(&paymentdomain.Payment{
		OrderID:         1,
		ChannelID:       10,
		ProviderType:    "official",
		ChannelType:     "alipay",
		InteractionMode: "qr",
		Currency:        "USD",
		Status:          "success",
		PaidAt:          &paidAt,
		ProviderPayload: jsonmap.JSON{paymentcontract.GatewayPayloadPaymentEnvironment: "live"},
	}); err != nil {
		t.Fatal(err)
	}
	liveDirectory := authorizationDirectory{&container.Container{OrderStore: store, PaymentStore: payments}}
	liveSnapshot, err := liveDirectory.Resolve(context.Background(), subscriptionauth.Subject{OrderNo: "member", UserID: 1})
	if err != nil || liveSnapshot.PaymentEnvironment != "live" {
		t.Fatalf("expected frozen live payment_environment, got %#v err=%v", liveSnapshot, err)
	}
}
