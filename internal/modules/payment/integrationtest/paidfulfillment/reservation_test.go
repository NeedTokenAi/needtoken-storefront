package paidfulfillment_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	productdomain "github.com/dujiao-next/internal/modules/catalog/product/domain"
	productstore "github.com/dujiao-next/internal/modules/catalog/product/store/gormstore"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	orderstore "github.com/dujiao-next/internal/modules/order/infrastructure/gormstore"
	paymentapp "github.com/dujiao-next/internal/modules/payment/application"
	paymentcontract "github.com/dujiao-next/internal/modules/payment/contract"
	paymentdomain "github.com/dujiao-next/internal/modules/payment/domain"
	paymentstore "github.com/dujiao-next/internal/modules/payment/infrastructure/gormstore"
	siteconnectiondomain "github.com/dujiao-next/internal/modules/siteconnection/domain"
	walletapp "github.com/dujiao-next/internal/modules/wallet/application"
	walletdomain "github.com/dujiao-next/internal/modules/wallet/domain"
	walletstore "github.com/dujiao-next/internal/modules/wallet/infrastructure/gormstore"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/dujiao-next/internal/shared/money"
	"github.com/dujiao-next/internal/upstream"
	"github.com/dujiao-next/internal/workflows/checkoutreservation"
	"github.com/dujiao-next/internal/workflows/paidfulfillment"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type reservationRemote struct {
	t                                                    *testing.T
	db                                                   *gorm.DB
	mu                                                   sync.Mutex
	capacity, posts, confirmPosts, queries               int
	loseReserve, loseConfirm, offline, late, quarantined bool
	ops                                                  map[string]string
	holds                                                map[string]paymentcontract.ReservationResult
}

func (s *reservationRemote) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	timestamp, _ := strconv.ParseInt(r.Header.Get(upstream.HeaderTimestamp), 10, 64)
	if r.URL.Path != "/api/v1/upstream/reservations" || r.Header.Get(upstream.HeaderApiKey) != "reservation-test-key" || !upstream.Verify("reservation-test-secret", r.Method, r.URL.Path, r.Header.Get(upstream.HeaderSignature), timestamp, body) {
		s.t.Error("unsigned reservation request")
		w.WriteHeader(401)
		return
	}
	// SQLite's single connection makes this a regression check for accidentally
	// holding a transaction open over a signed upstream HTTP request.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var n int64
	if err := s.db.WithContext(ctx).Model(&orderdomain.Order{}).Count(&n).Error; err != nil {
		s.t.Error("HTTP ran while DB connection was locked", err)
		w.WriteHeader(503)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.offline {
		w.WriteHeader(503)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error_code": "reservation_unavailable"})
		return
	}
	var request paymentcontract.ReservationRequest
	if r.Method == http.MethodGet {
		s.queries++
		operationID := r.URL.Query().Get("operation_id")
		hash, ok := s.ops[operationID]
		if !ok {
			w.WriteHeader(404)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error_code": "operation_not_found"})
			return
		}
		result := s.holds[r.URL.Query().Get("order_ref")]
		result.OperationID, result.PayloadHash = operationID, hash
		if s.late && result.Status == "reserved" {
			result.Status = "expired"
			result.ReadyForPayment = false
		}
		if s.quarantined && result.Status == "reserved" {
			result.Status = "needs_attention"
			result.ReadyForPayment = false
		}
		_ = json.NewEncoder(w).Encode(result)
		return
	}
	if r.Method != http.MethodPost || json.Unmarshal(body, &request) != nil || request.PayloadHash != paymentcontract.ReservationDigest(request) {
		w.WriteHeader(400)
		return
	}
	if old, ok := s.ops[request.OperationID]; ok {
		if old != request.PayloadHash {
			w.WriteHeader(409)
			return
		}
		result := s.holds[request.OrderRef]
		result.OperationID, result.PayloadHash = request.OperationID, old
		_ = json.NewEncoder(w).Encode(result)
		return
	}
	switch request.Action {
	case "reserve":
		if s.capacity == 0 {
			w.WriteHeader(409)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error_code": "no_stock"})
			return
		}
		s.capacity--
		s.posts++
		result := paymentcontract.ReservationResult{OK: true, OperationID: request.OperationID, PayloadHash: request.PayloadHash, OrderRef: request.OrderRef, HoldID: fmt.Sprintf("hold_remote_%03d", s.posts), Version: "1", Status: "reserved", BookStatus: "reserved", ReadyForPayment: true, BookOnly: true, EntitlementStatus: "unverified", Snapshot: request.Snapshot, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
		s.ops[request.OperationID] = request.PayloadHash
		s.holds[request.OrderRef] = result
		if s.loseReserve {
			s.loseReserve = false
			w.WriteHeader(503)
			return
		}
		_ = json.NewEncoder(w).Encode(result)
	case "confirm_payment":
		result, ok := s.holds[request.OrderRef]
		if !ok {
			w.WriteHeader(409)
			return
		}
		if request.ExpectedVersion != result.Version {
			w.WriteHeader(409)
			return
		}
		s.confirmPosts++
		result.Version = "2"
		result.OperationID, result.PayloadHash = request.OperationID, request.PayloadHash
		result.ReadyForPayment = false
		now := time.Now().UTC()
		result.PaidAt = &now
		if s.late || s.quarantined {
			result.Status, result.BookStatus = "needs_attention", "needs_attention"
		} else {
			result.Status, result.BookStatus = "paid", "paid"
		}
		s.ops[request.OperationID] = request.PayloadHash
		s.holds[request.OrderRef] = result
		if s.loseConfirm {
			s.loseConfirm = false
			w.WriteHeader(503)
			return
		}
		_ = json.NewEncoder(w).Encode(result)
	default:
		w.WriteHeader(400)
	}
}

type reservationProvider struct {
	connection paymentcontract.ReservationConnection
}

func (p *reservationProvider) OpenReservation(id uint) (paymentcontract.ReservationConnection, error) {
	if id != 7 {
		return nil, paymentcontract.ErrReservationUnavailable
	}
	return p.connection, nil
}

type checkoutFixture struct {
	db       *gorm.DB
	payments *paymentstore.Store
	service  *paymentapp.PaymentService
	gate     *checkoutreservation.Service
	remote   *reservationRemote
	provider *reservationProvider
	order    *orderdomain.Order
}

func checkout(t *testing.T, capacity int) *checkoutFixture {
	t.Helper()
	db, payments, _, order, _ := fixture(t)
	if err := db.AutoMigrate(&paymentdomain.CheckoutReservation{}, &productdomain.ProductSKU{}, &walletdomain.Account{}, &walletdomain.Transaction{}, &walletdomain.RechargeOrder{}); err != nil {
		t.Fatal(err)
	}
	expires := time.Now().Add(time.Hour)
	if err := db.Model(order).Updates(map[string]any{"user_id": uint(1), "currency": "CNY", "expires_at": expires}).Error; err != nil {
		t.Fatal(err)
	}
	order.UserID, order.Currency, order.ExpiresAt = 1, "CNY", &expires
	snapshot := jsonmap.JSON{"reservation_policy_known": true, "requires_reservation": true, "reservation_contract": jsonmap.JSON{"version": 1, "sku": "ChatGPT-Plus-US", "region": "US", "currency": "USD", "required_minor": "2000", "ttl_seconds": 900}, "reservation_connection_id": "7", "reservation_upstream_sku_id": "99"}
	if err := db.Model(&orderdomain.OrderItem{}).Where("order_id = ?", order.ID).Update("sku_snapshot_json", snapshot).Error; err != nil {
		t.Fatal(err)
	}
	order.Items[0].SKUSnapshotJSON = snapshot
	if err := db.Create(&productdomain.ProductSKU{ID: 1, ProductID: 1, SKUCode: "ChatGPT-Plus-US", RequiresReservation: true, ReservationPolicyKnown: true}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&walletdomain.Account{UserID: 1, Balance: money.FromDecimal(decimal.NewFromInt(500))}).Error; err != nil {
		t.Fatal(err)
	}
	remote := &reservationRemote{t: t, db: db, capacity: capacity, ops: map[string]string{}, holds: map[string]paymentcontract.ReservationResult{}}
	server := httptest.NewServer(remote)
	t.Cleanup(server.Close)
	adapter := upstream.NewDujiaoNextAdapter(&siteconnectiondomain.Connection{BaseURL: server.URL, ApiKey: "reservation-test-key", ApiSecret: "reservation-test-secret"}, "")
	provider := &reservationProvider{adapter}
	gate := checkoutreservation.New(payments, provider)
	wallets := walletstore.New(db)
	service := paymentapp.NewPaymentService(paymentapp.PaymentServiceOptions{OrderStore: orderstore.New(db, "outbox-test-credential-secret"), ProductSKURepo: productstore.NewSKUStore(db), PaymentStore: payments, WalletRepo: wallets, WalletService: walletapp.NewService(walletapp.Options{Repository: wallets, Transactions: wallets}), CheckoutReservations: gate})
	return &checkoutFixture{db: db, payments: payments, service: service, gate: gate, remote: remote, provider: provider, order: order}
}

func (f *checkoutFixture) balance(t *testing.T) string {
	t.Helper()
	var a walletdomain.Account
	if err := f.db.Where("user_id = 1").First(&a).Error; err != nil {
		t.Fatal(err)
	}
	return a.Balance.StringFixed(2)
}
func (f *checkoutFixture) rows(t *testing.T) []paymentdomain.CheckoutReservation {
	t.Helper()
	rows, err := f.payments.ListCheckoutReservations(f.order.ID)
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

func TestCheckoutReservationFailClosedBeforeWalletOrPayment(t *testing.T) {
	f := checkout(t, 0)
	var before int64
	f.db.Model(&paymentdomain.Payment{}).Count(&before)
	_, err := f.service.CreatePayment(paymentapp.CreatePaymentInput{OrderID: f.order.ID, UseBalance: true})
	if !errors.Is(err, paymentapp.ErrCheckoutReservationUnavailable) {
		t.Fatalf("missing reservation did not deny payment: %v", err)
	}
	var after int64
	f.db.Model(&paymentdomain.Payment{}).Count(&after)
	if before != after || f.balance(t) != "500.00" {
		t.Fatal("payment/debit occurred before authoritative hold")
	}
	rows := f.rows(t)
	if len(rows) != 1 || rows[0].State != "needs_attention" || rows[0].HoldID != "" {
		t.Fatal(rows)
	}
}

func TestCheckoutLostReserveResponseQueriesSameOperationThenWalletPays(t *testing.T) {
	f := checkout(t, 1)
	f.remote.loseReserve = true
	if _, err := f.service.CreatePayment(paymentapp.CreatePaymentInput{OrderID: f.order.ID, UseBalance: true}); !errors.Is(err, paymentapp.ErrCheckoutReservationUnavailable) {
		t.Fatal(err)
	}
	row := f.rows(t)[0]
	if row.State != "unknown" || f.balance(t) != "500.00" {
		t.Fatal(row)
	}
	firstID, firstHash := row.ReserveOperationID, row.ReservePayloadHash
	result, err := f.service.CreatePayment(paymentapp.CreatePaymentInput{OrderID: f.order.ID, UseBalance: true})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OrderPaid || f.balance(t) != "400.00" {
		t.Fatal("recovered reservation did not allow one wallet debit")
	}
	row = f.rows(t)[0]
	if row.ReserveOperationID != firstID || row.ReservePayloadHash != firstHash || f.remote.posts != 1 || f.remote.queries < 2 {
		t.Fatal("timeout allocated fresh stock", row)
	}
	var event paymentdomain.PaidFulfillmentEvent
	if err = f.db.First(&event, "order_id = ?", f.order.ID).Error; err != nil {
		t.Fatal("wallet payment lost durable event", err)
	}
	var item orderdomain.OrderItem
	f.db.First(&item, f.order.Items[0].ID)
	if item.SKUSnapshotJSON["reservation_hold"] == nil {
		t.Fatal("hold reference not snapshotted")
	}
}

func TestCheckoutUnavailableQueryNeverPostsANewHold(t *testing.T) {
	f := checkout(t, 1)
	f.remote.offline = true
	for range 2 {
		if _, err := f.service.CreatePayment(paymentapp.CreatePaymentInput{OrderID: f.order.ID, UseBalance: true}); !errors.Is(err, paymentapp.ErrCheckoutReservationUnavailable) {
			t.Fatal(err)
		}
	}
	if f.remote.posts != 0 || f.balance(t) != "500.00" {
		t.Fatal("unknown GET was treated as not found")
	}
}

func TestCheckoutFailedObservationCannotReuseCachedReadyHold(t *testing.T) {
	f := checkout(t, 1)
	if err := f.gate.EnsureBeforePayment(context.Background(), f.order.ID); err != nil {
		t.Fatal(err)
	}
	known := f.rows(t)[0]
	f.remote.offline = true
	if _, err := f.service.CreatePayment(paymentapp.CreatePaymentInput{OrderID: f.order.ID, UseBalance: true}); !errors.Is(err, paymentapp.ErrCheckoutReservationUnavailable) {
		t.Fatal(err)
	}
	row := f.rows(t)[0]
	if row.State != "unknown" || row.ReadyForPayment || row.HoldID != known.HoldID || row.HoldVersion != known.HoldVersion || row.AttentionCode == "" || f.balance(t) != "500.00" {
		t.Fatal("cached ready result hid failed observation", row)
	}
	var item orderdomain.OrderItem
	f.db.First(&item, f.order.Items[0].ID)
	receipt := item.SKUSnapshotJSON["reservation_hold"].(map[string]any)
	if receipt["state"] != "unknown" {
		t.Fatal("order snapshot concealed unknown observation", receipt)
	}
	f.remote.offline = false
	if _, err := f.service.CreatePayment(paymentapp.CreatePaymentInput{OrderID: f.order.ID, UseBalance: true}); err != nil {
		t.Fatal(err)
	}
	if f.remote.posts != 1 || f.balance(t) != "400.00" {
		t.Fatal("readback recovery replaced original hold")
	}
}

func TestCheckoutLastCapacityRaceAcrossOrders(t *testing.T) {
	f := checkout(t, 1)
	if f.db.Dialector.Name() == "postgres" {
		raw, _ := f.db.DB()
		raw.SetMaxOpenConns(5)
	}
	second := *f.order
	second.ID = 0
	second.OrderNo = "second-scarce-order"
	second.Items = nil
	if err := f.db.Create(&second).Error; err != nil {
		t.Fatal(err)
	}
	item := f.order.Items[0]
	item.ID = 0
	item.OrderID = second.ID
	if err := f.db.Create(&item).Error; err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, id := range []uint{f.order.ID, second.ID} {
		wg.Add(1)
		go func(i int, id uint) { defer wg.Done(); errs[i] = f.gate.EnsureBeforePayment(context.Background(), id) }(i, id)
	}
	wg.Wait()
	winners, denied := 0, 0
	for _, err := range errs {
		if err == nil {
			winners++
		} else {
			denied++
		}
	}
	if winners != 1 || denied != 1 || f.remote.posts != 1 || f.balance(t) != "500.00" {
		t.Fatalf("last capacity race: %v posts=%d", errs, f.remote.posts)
	}
}

func TestCheckoutConnectionIdentityCannotChangeAfterUnknownReserve(t *testing.T) {
	f := checkout(t, 1)
	f.remote.loseReserve = true
	if _, err := f.service.CreatePayment(paymentapp.CreatePaymentInput{OrderID: f.order.ID, UseBalance: true}); !errors.Is(err, paymentapp.ErrCheckoutReservationUnavailable) {
		t.Fatal(err)
	}
	row := f.rows(t)[0]
	if row.UpstreamIdentity == "" {
		t.Fatal("supplier identity not pinned before network")
	}
	queries, posts := f.remote.queries, f.remote.posts
	f.provider.connection = upstream.NewDujiaoNextAdapter(&siteconnectiondomain.Connection{BaseURL: "https://other-supplier.invalid", ApiKey: "different-merchant-key", ApiSecret: "different-secret"}, "")
	if _, err := f.service.CreatePayment(paymentapp.CreatePaymentInput{OrderID: f.order.ID, UseBalance: true}); !errors.Is(err, paymentapp.ErrCheckoutReservationUnavailable) {
		t.Fatal(err)
	}
	if f.rows(t)[0].State != "needs_attention" || f.remote.queries != queries || f.remote.posts != posts || f.balance(t) != "500.00" {
		t.Fatal("unknown operation moved to a different supplier/namespace")
	}
}

func TestCheckoutOrdinaryKnownPolicyKeepsExistingWalletFlow(t *testing.T) {
	f := checkout(t, 0)
	f.remote.offline = true
	if err := f.db.Model(&productdomain.ProductSKU{}).Where("id = 1").Update("requires_reservation", false).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.db.Model(&orderdomain.OrderItem{}).Where("order_id = ?", f.order.ID).Update("sku_snapshot_json", jsonmap.JSON{"reservation_policy_known": true, "requires_reservation": false}).Error; err != nil {
		t.Fatal(err)
	}
	r, err := f.service.CreatePayment(paymentapp.CreatePaymentInput{OrderID: f.order.ID, UseBalance: true})
	if err != nil || !r.OrderPaid {
		t.Fatal("ordinary product unexpectedly requires upstream hold", err)
	}
	if f.remote.posts != 0 || f.balance(t) != "400.00" {
		t.Fatal("ordinary flow contacted reservation book")
	}
}

func TestCheckoutExistingPaymentCallbackRecordsMoneyButNeverAllocatesMissingHold(t *testing.T) {
	f := checkout(t, 1)
	var payment paymentdomain.Payment
	if err := f.db.Where("order_id = ?", f.order.ID).First(&payment).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.db.Model(&payment).Update("currency", "CNY").Error; err != nil {
		t.Fatal(err)
	}
	payment.Currency = "CNY"
	if _, err := f.service.HandleCallback(paymentapp.PaymentCallbackInput{PaymentID: payment.ID, Status: "success", Amount: payment.Amount, Currency: "CNY"}); err != nil {
		t.Fatal(err)
	}
	var stored orderdomain.Order
	f.db.First(&stored, f.order.ID)
	if stored.PaidAt == nil {
		t.Fatal("actual payment fact was discarded")
	}
	creator := &creator{}
	if err := paidfulfillment.New(f.payments, creator, f.service).RecoverPaidFulfillment(); !errors.Is(err, paymentcontract.ErrReservationAttention) {
		t.Fatal("missing hold after payment did not require review", err)
	}
	if creator.calls != 0 || f.remote.posts != 0 || len(f.rows(t)) != 0 {
		t.Fatal("paid recovery allocated a fresh hold")
	}
}

func TestCheckoutMissingOutboxConfirmerFailsClosed(t *testing.T) {
	f := checkout(t, 1)
	if _, err := f.service.CreatePayment(paymentapp.CreatePaymentInput{OrderID: f.order.ID, UseBalance: true}); err != nil {
		t.Fatal(err)
	}
	creator := &creator{}
	if err := paidfulfillment.New(f.payments, creator).RecoverPaidFulfillment(); !errors.Is(err, paymentcontract.ErrReservationUnavailable) {
		t.Fatal("unwired confirmer silently authorized procurement", err)
	}
	if creator.calls != 0 {
		t.Fatal("unconfirmed paid hold procured")
	}
}

func TestCheckoutLatePaidHoldStopsProcurementWithoutReallocation(t *testing.T) {
	f := checkout(t, 1)
	if _, err := f.service.CreatePayment(paymentapp.CreatePaymentInput{OrderID: f.order.ID, UseBalance: true}); err != nil {
		t.Fatal(err)
	}
	f.remote.late = true
	creator := &creator{}
	if err := paidfulfillment.New(f.payments, creator, f.service).RecoverPaidFulfillment(); !errors.Is(err, paymentcontract.ErrReservationAttention) {
		t.Fatalf("late hold not isolated: %v", err)
	}
	if creator.calls != 0 || f.remote.posts != 1 || f.remote.confirmPosts != 1 {
		t.Fatal("late payment created fresh procurement/stock")
	}
	row := f.rows(t)[0]
	if row.State != "needs_attention" || row.HoldID == "" {
		t.Fatal(row)
	}
	var event paymentdomain.PaidFulfillmentEvent
	f.db.First(&event, "order_id = ?", f.order.ID)
	if event.CompletedAt != nil {
		t.Fatal("attention event incorrectly acknowledged")
	}
}

func TestCheckoutLostPaidConfirmationRecoversBeforeProcurement(t *testing.T) {
	f := checkout(t, 1)
	if _, err := f.service.CreatePayment(paymentapp.CreatePaymentInput{OrderID: f.order.ID, UseBalance: true}); err != nil {
		t.Fatal(err)
	}
	f.remote.loseConfirm = true
	creator := &creator{}
	if err := paidfulfillment.New(f.payments, creator, f.service).RecoverPaidFulfillment(); err == nil {
		t.Fatal("lost response should remain unknown")
	}
	if creator.calls != 0 {
		t.Fatal("unknown confirm started procurement")
	}
	f.db.Model(&paymentdomain.PaidFulfillmentEvent{}).Where("order_id = ?", f.order.ID).Update("next_attempt_at", time.Now().Add(-time.Second))
	if err := paidfulfillment.New(f.payments, creator, f.service).RecoverPaidFulfillment(); err != nil {
		t.Fatal(err)
	}
	if creator.calls != 1 || f.remote.confirmPosts != 1 || f.remote.posts != 1 {
		t.Fatal("confirmation replay duplicated effect")
	}
	row := f.rows(t)[0]
	if row.State != "paid" || row.HoldVersion != "2" {
		t.Fatal(row)
	}
}

func TestCheckoutLegacyUnknownPolicyAndBuyerFormCannotBypassGate(t *testing.T) {
	f := checkout(t, 1)
	for _, snapshot := range []jsonmap.JSON{nil, {"reservation_policy_known": false, "requires_reservation": false}} {
		if err := f.db.Model(&orderdomain.OrderItem{}).Where("order_id = ?", f.order.ID).Updates(map[string]any{"sku_snapshot_json": snapshot, "manual_form_submission_json": jsonmap.JSON{"reservation_policy_known": true, "requires_reservation": false}}).Error; err != nil {
			t.Fatal(err)
		}
		if _, err := f.service.CreatePayment(paymentapp.CreatePaymentInput{OrderID: f.order.ID, UseBalance: true}); !errors.Is(err, paymentapp.ErrCheckoutReservationUnavailable) {
			t.Fatal("legacy unverified policy was sold", err)
		}
	}
	if f.remote.posts != 0 || f.balance(t) != "500.00" {
		t.Fatal("buyer form or current SKU guessed a historical promise")
	}
}

func TestReservationWireDigestGoldenFixture(t *testing.T) {
	r := paymentcontract.ReservationRequest{OperationID: "sf_reserve_001", Action: "reserve", OrderRef: "ORDER_1", Snapshot: paymentcontract.ReservationSnapshot{SKU: "ChatGPT-Plus-US", Region: "US", Currency: "USD", RequiredMinor: "2000", ExpiresAt: "2026-11-03T12:00:00Z"}, ExpectedVersion: "0"}
	if got := paymentcontract.ReservationDigest(r); got != "b62a0be0817f6de601cfcd61778a81dcc76ab31c1ce51c03cbc414e95d56d5c6" {
		t.Fatal(got)
	}
}

// Optional cross-repository smoke against an isolated glue test server seeded
// with matching capacity. It never points at a configured production service.
func TestCheckoutRealGlueEndpoint(t *testing.T) {
	base := os.Getenv("NEEDTOKEN_TEST_GLUE_RESERVATION_URL")
	if base == "" {
		t.Skip("isolated real glue reservation test server is not set")
	}
	key, secret := os.Getenv("NEEDTOKEN_TEST_GLUE_RESERVATION_KEY"), os.Getenv("NEEDTOKEN_TEST_GLUE_RESERVATION_SECRET")
	if key == "" || secret == "" {
		t.Fatal("test-only glue credentials required")
	}
	f := checkout(t, 0)
	// Different DB engine/runs use different merchant order numbers without
	// changing the production deterministic per-order operation algorithm.
	if err := f.db.Model(f.order).Update("order_no", fmt.Sprintf("real-glue-smoke-%d", time.Now().UnixNano())).Error; err != nil {
		t.Fatal(err)
	}
	f.provider.connection = upstream.NewDujiaoNextAdapter(&siteconnectiondomain.Connection{BaseURL: base, ApiKey: key, ApiSecret: secret}, "")
	result, err := f.service.CreatePayment(paymentapp.CreatePaymentInput{OrderID: f.order.ID, UseBalance: true})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OrderPaid || f.balance(t) != "400.00" {
		t.Fatal("real signed hold did not permit one wallet payment")
	}
	creator := &creator{}
	if err := paidfulfillment.New(f.payments, creator, f.service).RecoverPaidFulfillment(); err != nil {
		t.Fatal(err)
	}
	row := f.rows(t)[0]
	if creator.calls != 1 || row.State != "paid" || row.HoldID == "" || row.HoldVersion != "2" {
		t.Fatal("real native paid hold did not precede procurement", row)
	}
	request := paymentcontract.ReservationRequest{}
	raw, _ := json.Marshal(row.ConfirmRequestJSON)
	_ = json.Unmarshal(raw, &request)
	current, err := f.provider.connection.GetReservation(context.Background(), row.OrderRef, request.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Status != "paid" || current.PayloadHash != request.PayloadHash || current.ReadyForPayment {
		t.Fatal("real hold status/hash disagree", current)
	}
}
