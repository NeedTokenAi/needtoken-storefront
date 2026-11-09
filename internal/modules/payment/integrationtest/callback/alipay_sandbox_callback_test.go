package paymentcallback_test

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/dujiao-next/internal/constants"
	fulfillmentdomain "github.com/dujiao-next/internal/modules/fulfillment/domain"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	orderstore "github.com/dujiao-next/internal/modules/order/infrastructure/gormstore"
	paymentapp "github.com/dujiao-next/internal/modules/payment/application"
	paymentdomain "github.com/dujiao-next/internal/modules/payment/domain"
	alipayadapter "github.com/dujiao-next/internal/modules/payment/infrastructure/gateway/adapters/alipay"
	paymentprovider "github.com/dujiao-next/internal/modules/payment/infrastructure/gateway/provider"
	paymentstore "github.com/dujiao-next/internal/modules/payment/infrastructure/gormstore"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/dujiao-next/internal/shared/money"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// sandboxAlipayGateway is an explicit non-production stand-in. Live Alipay
// sandbox credentials were not available, and this test does not dial any host.
const sandboxAlipayGateway = "https://sandbox.example.test/gateway.do"

func TestAlipaySandboxSignedCallbackIsIdempotent(t *testing.T) {
	if strings.Contains(sandboxAlipayGateway, "openapi.alipay.com") {
		t.Fatal("sandbox callback must not use the production Alipay gateway")
	}
	db := sandboxDB(t)
	amount := money.FromDecimal(decimal.RequireFromString("88.00"))
	order := &orderdomain.Order{OrderNo: "DJ-SANDBOX-1", Status: constants.OrderStatusPendingPayment, Currency: "CNY", TotalAmount: amount, OriginalAmount: amount, Items: []orderdomain.OrderItem{{ProductID: 1, SKUID: 1, Quantity: 1, TitleJSON: jsonmap.JSON{"zh-CN": "API plan"}, FulfillmentType: constants.FulfillmentTypeUpstream, TotalPrice: amount, UnitPrice: amount}}}
	if err := db.Create(order).Error; err != nil {
		t.Fatal(err)
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	privateDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	publicDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	channel := &paymentdomain.PaymentChannel{
		Name: "Alipay sandbox", ProviderType: constants.PaymentProviderOfficial, ChannelType: constants.PaymentChannelTypeAlipay,
		InteractionMode: constants.PaymentInteractionQR, FeeRate: money.FromDecimal(decimal.Zero), IsActive: true,
		ConfigJSON: jsonmap.JSON{
			"app_id": "2021000000000000", "sign_type": "RSA2", "gateway_url": sandboxAlipayGateway,
			"notify_url":        "https://store.example.test/api/v1/payments/callback",
			"private_key":       string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateDER})),
			"alipay_public_key": string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER})),
		},
	}
	if err := db.Create(channel).Error; err != nil {
		t.Fatal(err)
	}
	if got, _ := channel.ConfigJSON["gateway_url"].(string); got != sandboxAlipayGateway {
		t.Fatalf("gateway = %s", got)
	}
	payment := &paymentdomain.Payment{OrderID: order.ID, ChannelID: channel.ID, ProviderType: channel.ProviderType, ChannelType: channel.ChannelType, InteractionMode: channel.InteractionMode, Amount: amount, Currency: "CNY", Status: constants.PaymentStatusPending, GatewayOrderNo: "DJP-SANDBOX-1"}
	if err := db.Create(payment).Error; err != nil {
		t.Fatal(err)
	}
	newService := func() *paymentapp.PaymentService {
		registry := paymentprovider.NewRegistry()
		registry.Register(constants.PaymentProviderOfficial, constants.PaymentChannelTypeAlipay, alipayadapter.NewAlipayAdapter())
		return paymentapp.NewPaymentService(paymentapp.PaymentServiceOptions{
			OrderStore: orderstore.New(db, "sandbox-callback-credential-secret"), PaymentStore: paymentstore.New(db, "sandbox-callback-credential-secret"),
			ChannelStore: paymentstore.NewChannelStore(db), PaymentProviderRegistry: registry,
		})
	}
	service := newService()
	form := map[string][]string{
		"app_id": {channel.ConfigJSON["app_id"].(string)}, "notify_id": {"sandbox-notify-1"}, "notify_type": {"trade_status_sync"},
		"out_trade_no": {payment.GatewayOrderNo}, "trade_no": {"2026110900000001"}, "trade_status": {"TRADE_SUCCESS"},
		"total_amount": {"88.00"}, "gmt_payment": {"2026-11-09 12:00:00"}, "sign_type": {"RSA2"},
	}
	if _, err := service.HandleSyncCallback(channel, form, nil); err == nil {
		t.Fatal("unsigned sandbox callback was accepted")
	}
	if countPaidEvents(t, db, order.ID) != 0 {
		t.Fatal("rejected callback created a paid event")
	}
	form["sign"] = []string{sandboxRSA2Sign(t, form, key)}
	for range 2 {
		paid, err := service.HandleSyncCallback(channel, form, nil)
		if err != nil || paid.Status != constants.PaymentStatusSuccess || paid.ProviderRef != "2026110900000001" {
			t.Fatalf("signed callback: %+v %v", paid, err)
		}
	}
	restarted := newService()
	paid, err := restarted.HandleSyncCallback(channel, form, nil)
	if err != nil || paid.ProviderRef != "2026110900000001" {
		t.Fatalf("restarted callback: %+v %v", paid, err)
	}
	if countPaidEvents(t, db, order.ID) != 1 {
		t.Fatalf("paid events = %d, want 1", countPaidEvents(t, db, order.ID))
	}
}

func sandboxDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:alipay_sandbox_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	raw.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = raw.Close() })
	if err := db.AutoMigrate(&orderdomain.Order{}, &orderdomain.OrderItem{}, &fulfillmentdomain.Fulfillment{}, &paymentdomain.PaymentChannel{}, &paymentdomain.Payment{}, &paymentdomain.PaidFulfillmentEvent{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func countPaidEvents(t *testing.T, db *gorm.DB, orderID uint) int64 {
	t.Helper()
	var count int64
	if err := db.Model(&paymentdomain.PaidFulfillmentEvent{}).Where("order_id = ?", orderID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	return count
}

func sandboxRSA2Sign(t *testing.T, form map[string][]string, key *rsa.PrivateKey) string {
	t.Helper()
	sum := sha256.Sum256([]byte(sandboxCanonical(form)))
	signed, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(signed)
}

func sandboxCanonical(form map[string][]string) string {
	keys := make([]string, 0, len(form))
	for key, values := range form {
		if len(values) == 0 || strings.TrimSpace(values[0]) == "" {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" || strings.EqualFold(key, "sign") || strings.EqualFold(key, "sign_type") {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+form[key][0])
	}
	return strings.Join(parts, "&")
}
