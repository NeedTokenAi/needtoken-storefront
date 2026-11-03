package paidfulfillment_test

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dujiao-next/internal/constants"
	fulfillmentdomain "github.com/dujiao-next/internal/modules/fulfillment/domain"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	orderstore "github.com/dujiao-next/internal/modules/order/infrastructure/gormstore"
	paymentapp "github.com/dujiao-next/internal/modules/payment/application"
	paymentdomain "github.com/dujiao-next/internal/modules/payment/domain"
	paymentstore "github.com/dujiao-next/internal/modules/payment/infrastructure/gormstore"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/dujiao-next/internal/shared/money"
	"github.com/dujiao-next/internal/workflows/paidfulfillment"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type creator struct {
	calls int
	fail  bool
}

func (c *creator) CreateForOrder(uint) error {
	c.calls++
	if c.fail {
		return errors.New("procurement storage unavailable")
	}
	return nil
}
func fixture(t *testing.T) (*gorm.DB, *paymentstore.Store, *paymentapp.PaymentService, *orderdomain.Order, *paymentdomain.Payment) {
	t.Helper()
	config := &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true}
	var db *gorm.DB
	var err error
	if postgresDSN := os.Getenv("NEEDTOKEN_STOREFRONT_TEST_POSTGRES_DSN"); postgresDSN != "" {
		// Each test owns a fresh schema. The supplied database's existing schemas,
		// credentials and production data are never migrated or cleared.
		admin, openErr := gorm.Open(postgres.Open(postgresDSN), config)
		if openErr != nil {
			t.Fatal(openErr)
		}
		schema := fmt.Sprintf("nt_storefront_outbox_%d", time.Now().UnixNano())
		if err := admin.Exec("CREATE SCHEMA " + schema).Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := admin.Exec("DROP SCHEMA " + schema + " CASCADE").Error; err != nil {
				t.Error(err)
			}
			raw, _ := admin.DB()
			if raw != nil {
				_ = raw.Close()
			}
		})
		parsed, parseErr := url.Parse(postgresDSN)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		query := parsed.Query()
		query.Set("search_path", schema)
		parsed.RawQuery = query.Encode()
		db, err = gorm.Open(postgres.Open(parsed.String()), config)
	} else {
		db, err = gorm.Open(sqlite.Open(fmt.Sprintf("file:outbox_%d?mode=memory&cache=shared", time.Now().UnixNano())), config)
	}
	if err != nil {
		t.Fatal(err)
	}
	raw, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	raw.SetMaxOpenConns(1)
	if db.Dialector.Name() == "postgres" {
		var version string
		if err := db.Raw("SELECT current_setting('server_version')").Scan(&version).Error; err != nil {
			t.Fatal(err)
		}
		t.Logf("financial integration database: PostgreSQL %s (isolated schema)", version)
	}
	t.Cleanup(func() { _ = raw.Close() })
	if err := db.AutoMigrate(&orderdomain.Order{}, &orderdomain.OrderItem{}, &fulfillmentdomain.Fulfillment{}, &paymentdomain.Payment{}, &paymentdomain.PaidFulfillmentEvent{}); err != nil {
		t.Fatal(err)
	}
	amount := money.FromDecimal(decimal.NewFromInt(100))
	order := &orderdomain.Order{OrderNo: "outbox-order", Status: constants.OrderStatusPendingPayment, Currency: "USD", TotalAmount: amount, OriginalAmount: amount, Items: []orderdomain.OrderItem{{ProductID: 1, SKUID: 1, Quantity: 1, TitleJSON: jsonmap.JSON{"en-US": "Upstream product"}, FulfillmentType: constants.FulfillmentTypeUpstream, TotalPrice: amount, UnitPrice: amount}}}
	if err := db.Create(order).Error; err != nil {
		t.Fatal(err)
	}
	payment := &paymentdomain.Payment{OrderID: order.ID, Status: constants.PaymentStatusPending, Amount: amount, Currency: "USD"}
	if err := db.Create(payment).Error; err != nil {
		t.Fatal(err)
	}
	payments := paymentstore.New(db, "outbox-test-credential-secret")
	service := paymentapp.NewPaymentService(paymentapp.PaymentServiceOptions{OrderStore: orderstore.New(db, "outbox-test-credential-secret"), PaymentStore: payments})
	return db, payments, service, order, payment
}
func pay(service *paymentapp.PaymentService, payment *paymentdomain.Payment) error {
	_, err := service.HandleCallback(paymentapp.PaymentCallbackInput{PaymentID: payment.ID, Status: constants.PaymentStatusSuccess, Amount: payment.Amount, Currency: "USD"})
	return err
}
func TestPaymentAndOutboxRollbackTogether(t *testing.T) {
	db, _, service, order, payment := fixture(t)
	if err := db.Callback().Create().Before("gorm:create").Register("fail_outbox", func(tx *gorm.DB) {
		if tx.Statement.Table == "paid_fulfillment_events" {
			tx.AddError(errors.New("outbox disk failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	if err := pay(service, payment); err == nil {
		t.Fatal("payment must not commit without durable event")
	}
	var stored orderdomain.Order
	if err := db.First(&stored, order.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.PaidAt != nil || stored.Status != constants.OrderStatusPendingPayment {
		t.Fatalf("order payment leaked across rollback: %+v", stored)
	}
	var storedPayment paymentdomain.Payment
	db.First(&storedPayment, payment.ID)
	if storedPayment.Status != constants.PaymentStatusPending {
		t.Fatalf("payment committed without outbox: %s", storedPayment.Status)
	}
}
func TestQueueDisabledAndFailedDispatchRecoverAfterRestart(t *testing.T) {
	db, payments, service, order, payment := fixture(t)
	upstream := &creator{fail: true}
	service.SetProcurementService(upstream)
	if err := pay(service, payment); err != nil {
		t.Fatal(err)
	}
	if upstream.calls != 1 {
		t.Fatal("queue-disabled payment did not try procurement creation")
	}
	var event paymentdomain.PaidFulfillmentEvent
	if err := db.First(&event, "order_id = ?", order.ID).Error; err != nil {
		t.Fatal("paid event lost", err)
	}
	if err := paidfulfillment.New(payments, upstream).RecoverPaidFulfillment(); err == nil {
		t.Fatal("failed creation must remain retryable")
	}
	db.Model(&event).Update("next_attempt_at", time.Now().Add(-time.Second))
	upstream.fail = false
	if err := paidfulfillment.New(payments, upstream).RecoverPaidFulfillment(); err != nil {
		t.Fatal(err)
	}
	db.First(&event, "order_id = ?", order.ID)
	if event.CompletedAt == nil || event.Attempts != 2 {
		t.Fatalf("unexpected recovered event: %+v", event)
	}
	before := upstream.calls
	if err := paidfulfillment.New(payments, upstream).RecoverPaidFulfillment(); err != nil {
		t.Fatal(err)
	}
	if upstream.calls != before {
		t.Fatal("completed event redelivered")
	}
}
func TestHistoricalPaidGapBackfillAndLeaseRecovery(t *testing.T) {
	db, payments, _, order, _ := fixture(t)
	now := time.Now()
	if err := db.Model(order).Updates(map[string]interface{}{"status": constants.OrderStatusPaid, "paid_at": now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := payments.BackfillPaidFulfillment(now, 100); err != nil {
		t.Fatal(err)
	}
	claimed, err := payments.ClaimPaidFulfillment(order.ID, "old-process", now)
	if err != nil || !claimed {
		t.Fatal(claimed, err)
	}
	claimed, err = payments.ClaimPaidFulfillment(order.ID, "overlap", now)
	if err != nil || claimed {
		t.Fatal("live lease was stolen", claimed, err)
	}
	claimed, err = payments.ClaimPaidFulfillment(order.ID, "restarted", now.Add(6*time.Minute))
	if err != nil || !claimed {
		t.Fatal("expired lease not recovered", claimed, err)
	}
	if err := payments.FinishPaidFulfillment(order.ID, "old-process", true, now); err != nil {
		t.Fatal(err)
	}
	var event paymentdomain.PaidFulfillmentEvent
	db.First(&event, "order_id = ?", order.ID)
	if event.CompletedAt != nil {
		t.Fatal("expired claimant acknowledged newer lease")
	}
}

func TestCompetingPaymentCallbacksAndEventClaims(t *testing.T) {
	db, payments, service, order, payment := fixture(t)
	raw, _ := db.DB()
	if db.Dialector.Name() == "postgres" {
		raw.SetMaxOpenConns(8)
	}
	const contenders = 8
	var wg sync.WaitGroup
	failures := make(chan error, contenders)
	for range contenders {
		wg.Add(1)
		go func() { defer wg.Done(); failures <- pay(service, payment) }()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Error(err)
		}
	}
	var count int64
	if err := db.Model(&paymentdomain.PaidFulfillmentEvent{}).Where("order_id = ?", order.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("duplicate paid events: %d", count)
	}
	var claims atomic.Int32
	failures = make(chan error, contenders)
	for i := range contenders {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			claimed, err := payments.ClaimPaidFulfillment(order.ID, fmt.Sprintf("process-%d", i), time.Now())
			if claimed {
				claims.Add(1)
			}
			failures <- err
		}(i)
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Error(err)
		}
	}
	if claims.Load() != 1 {
		t.Fatalf("concurrent dispatch lease owners: %d", claims.Load())
	}
}

func TestOutboxMigrationUpgradeTwicePreservesPaidOrderAndEvents(t *testing.T) {
	db, payments, _, order, _ := fixture(t)
	// Simulate a pre-upgrade paid order with no outbox table yet, only inside
	// this test's isolated schema/in-memory SQLite database.
	if err := db.Migrator().DropTable(&paymentdomain.PaidFulfillmentEvent{}); err != nil {
		t.Fatal(err)
	}
	paidAt := time.Now().UTC().Truncate(time.Microsecond)
	if err := db.Model(order).Updates(map[string]interface{}{"status": constants.OrderStatusPaid, "paid_at": paidAt}).Error; err != nil {
		t.Fatal(err)
	}
	for iteration := 0; iteration < 2; iteration++ {
		if err := db.AutoMigrate(&orderdomain.Order{}, &orderdomain.OrderItem{}, &fulfillmentdomain.Fulfillment{}, &paymentdomain.Payment{}, &paymentdomain.PaidFulfillmentEvent{}); err != nil {
			t.Fatal(err)
		}
		if err := payments.BackfillPaidFulfillment(paidAt, 100); err != nil {
			t.Fatal(err)
		}
		if iteration == 0 {
			if err := db.Model(&paymentdomain.PaidFulfillmentEvent{}).Where("order_id = ?", order.ID).Update("attempts", 3).Error; err != nil {
				t.Fatal(err)
			}
		}
	}
	var stored orderdomain.Order
	if err := db.First(&stored, order.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != constants.OrderStatusPaid || stored.PaidAt == nil || !stored.PaidAt.Equal(paidAt) {
		t.Fatalf("migration changed payment facts: %+v", stored)
	}
	var event paymentdomain.PaidFulfillmentEvent
	if err := db.First(&event, "order_id = ?", order.ID).Error; err != nil {
		t.Fatal(err)
	}
	if event.Attempts != 3 || event.CompletedAt != nil {
		t.Fatalf("migration replaced existing event: %+v", event)
	}
	var count int64
	db.Model(&paymentdomain.PaidFulfillmentEvent{}).Count(&count)
	if count != 1 {
		t.Fatalf("repeated migration/backfill duplicated event: %d", count)
	}
}
