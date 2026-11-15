package subscriptionauthorizationstore

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	ordercontract "github.com/dujiao-next/internal/modules/order/contract"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testStore(t *testing.T) *SubscriptionAuthorizationStore {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:order_authorization_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sql, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sql.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sql.Close() })
	if err = db.AutoMigrate(&orderdomain.Order{}, &orderdomain.SubscriptionAuthorization{}); err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&orderdomain.Order{ID: 1, OrderNo: "ORDER-1", Status: "paid", Currency: "USD"}).Error; err != nil {
		t.Fatal(err)
	}
	return NewSubscriptionAuthorizationStore(db)
}
func TestCreateSerializesDistinctChallengeIDsAndPreservesReplay(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	var created atomic.Int32
	var wg sync.WaitGroup
	for n := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, fresh, err := s.Create(ctx, &orderdomain.SubscriptionAuthorization{ID: fmt.Sprintf("auth_%d", n), OrderID: 1, OrderNo: "ORDER-1", TenantRef: "main", BuyerRef: "guest:1", ContextHash: "same", Status: "awaiting_email_proof", CreatedAt: now, ExpiresAt: now.Add(30 * time.Minute)}, true)
			if err != nil && err != ordercontract.ErrAuthorizationRateLimited {
				t.Errorf("create: %v", err)
			}
			if fresh {
				created.Add(1)
			}
		}()
	}
	wg.Wait()
	if created.Load() != 1 {
		t.Fatalf("concurrent challenges sent: %d", created.Load())
	}
	var row orderdomain.SubscriptionAuthorization
	if err := s.db.First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if _, fresh, err := s.Create(ctx, &row, true); err != nil || fresh {
		t.Fatal("same operation was not replayed")
	}
	row.ContextHash = "changed"
	if _, _, err := s.Create(ctx, &row, true); err != ordercontract.ErrAuthorizationConflict {
		t.Fatal("operation snapshot changed")
	}
}
func TestUpdateCommitsAttemptAndRollsBackRejectedChanges(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	_, _, err := s.Create(ctx, &orderdomain.SubscriptionAuthorization{ID: "auth_1", OrderID: 1, ContextHash: "same", CreatedAt: now, ExpiresAt: now.Add(time.Minute)}, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Update(ctx, "auth_1", func(r *orderdomain.SubscriptionAuthorization) error { r.Attempts++; return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Update(ctx, "auth_1", func(r *orderdomain.SubscriptionAuthorization) error {
		r.Attempts++
		return ordercontract.ErrAuthorizationConflict
	}); err != ordercontract.ErrAuthorizationConflict {
		t.Fatal(err)
	}
	row, err := s.Get(ctx, "auth_1")
	if err != nil || row.Attempts != 1 {
		t.Fatal("attempt count changed incorrectly")
	}
}
