package procurement_test

import (
	"errors"
	procurementcontract "github.com/dujiao-next/internal/modules/procurement/contract"
	procurementstore "github.com/dujiao-next/internal/modules/procurement/infrastructure/gormstore"
	"testing"

	mappingdomain "github.com/dujiao-next/internal/modules/catalog/mapping/domain"

	"github.com/dujiao-next/internal/constants"
)

// ── CreateForOrder tests ──

func TestCreateForOrder_SkipsNonUpstreamItems(t *testing.T) {
	db := setupProcurementTestDB(t)

	order := createProcTestOrder(t, db, "PROC-SKIP-001", constants.OrderStatusPaid, constants.FulfillmentTypeAuto)

	connSvc := newTestSiteConnectionService(db, "test-key", t.TempDir())
	svc := newTestProcurementService(db, connSvc)

	if err := svc.CreateForOrder(order.ID); err != nil {
		t.Fatalf("CreateForOrder: %v", err)
	}

	// 验证没有创建采购单
	var count int64
	db.Model(&ProcurementOrder{}).Count(&count)
	if count != 0 {
		t.Errorf("expected no procurement orders for auto fulfillment, got %d", count)
	}
}

func TestCreateForOrder_IdempotentSkipsDuplicate(t *testing.T) {
	db := setupProcurementTestDB(t)

	order := createProcTestOrder(t, db, "PROC-DUP-001", constants.OrderStatusPaid, constants.FulfillmentTypeUpstream)
	pm := &mappingdomain.Mapping{ConnectionID: 1, LocalProductID: 1, UpstreamProductID: 101, IsActive: true}
	db.Create(pm)

	connSvc := newTestSiteConnectionService(db, "test-key", t.TempDir())
	svc := newTestProcurementService(db, connSvc)

	// 第一次创建成功
	if err := svc.CreateForOrder(order.ID); err != nil {
		t.Fatalf("first CreateForOrder: %v", err)
	}

	// 第二次应该返回 ErrExists
	err := svc.CreateForOrder(order.ID)
	if err != ErrExists {
		t.Errorf("expected ErrExists on duplicate, got: %v", err)
	}
}

func TestProcurementUniqueOrderConstraintSurvivesDuplicateWriters(t *testing.T) {
	db := setupProcurementTestDB(t)
	order := createProcTestOrder(t, db, "PROC-DB-UNIQUE", constants.OrderStatusPaid, constants.FulfillmentTypeUpstream)
	store := procurementstore.New(db)
	first := &ProcurementOrder{LocalOrderID: order.ID, ConnectionID: 1, Status: "pending", Currency: "USD"}
	if err := store.Create(first); err != nil {
		t.Fatal(err)
	}
	// Bypass application pre-check to exercise racing writers' database boundary.
	if err := store.Create(&ProcurementOrder{LocalOrderID: order.ID, ConnectionID: 1, Status: "pending", Currency: "USD"}); !errors.Is(err, procurementcontract.ErrExists) {
		t.Fatalf("expected unique business conflict, got %v", err)
	}
	var count int64
	db.Model(&ProcurementOrder{}).Where("local_order_id = ?", order.ID).Count(&count)
	if count != 1 {
		t.Fatalf("duplicate procurement count: %d", count)
	}
}
