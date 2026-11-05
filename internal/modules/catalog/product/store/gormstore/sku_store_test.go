package gormstore

import (
	"fmt"
	"gorm.io/driver/postgres"
	"net/url"
	"os"
	"testing"
	"time"

	productdomain "github.com/dujiao-next/internal/modules/catalog/product/domain"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/dujiao-next/internal/shared/money"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func setupSKUStoreTest(t *testing.T) (*SKUStore, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:product_sku_repository_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&productdomain.ProductSKU{}); err != nil {
		t.Fatalf("migrate product sku failed: %v", err)
	}
	return NewSKUStore(db), db
}

func TestSKUStoreListByProductSortOrderDescending(t *testing.T) {
	repo, _ := setupSKUStoreTest(t)

	high := &productdomain.ProductSKU{
		ProductID:      1,
		SKUCode:        "HIGH",
		PriceAmount:    money.FromDecimal(decimal.NewFromInt(100)),
		IsActive:       true,
		SortOrder:      100,
		SpecValuesJSON: jsonmap.JSON{},
	}
	low := &productdomain.ProductSKU{
		ProductID:      1,
		SKUCode:        "LOW",
		PriceAmount:    money.FromDecimal(decimal.NewFromInt(100)),
		IsActive:       true,
		SortOrder:      1,
		SpecValuesJSON: jsonmap.JSON{},
	}
	if err := repo.Create(high); err != nil {
		t.Fatalf("create high sort sku failed: %v", err)
	}
	if err := repo.Create(low); err != nil {
		t.Fatalf("create low sort sku failed: %v", err)
	}

	rows, err := repo.ListByProduct(1, true)
	if err != nil {
		t.Fatalf("list skus failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 skus, got %d", len(rows))
	}
	if rows[0].SKUCode != "HIGH" || rows[1].SKUCode != "LOW" {
		t.Fatalf("expected high sort_order first, got %s then %s", rows[0].SKUCode, rows[1].SKUCode)
	}
}

func TestProductSKUManualStockLifecycleMatchesProductSemantics(t *testing.T) {
	repo, db := setupSKUStoreTest(t)
	sku := &productdomain.ProductSKU{
		ProductID:        1,
		SKUCode:          "STOCK-LIFECYCLE",
		PriceAmount:      money.FromDecimal(decimal.NewFromInt(100)),
		ManualStockTotal: 10,
		IsActive:         true,
	}
	if err := repo.Create(sku); err != nil {
		t.Fatalf("create stock sku: %v", err)
	}

	assertAffected := func(operation string, affected int64, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s manual stock: %v", operation, err)
		}
		if affected != 1 {
			t.Fatalf("%s affected rows want 1 got %d", operation, affected)
		}
	}

	affected, err := repo.ReserveManualStock(sku.ID, 3)
	assertAffected("reserve", affected, err)
	affected, err = repo.ConsumeManualStock(sku.ID, 2)
	assertAffected("consume", affected, err)
	affected, err = repo.ReleaseManualStock(sku.ID, 1)
	assertAffected("release", affected, err)

	var reloaded productdomain.ProductSKU
	if err := db.First(&reloaded, sku.ID).Error; err != nil {
		t.Fatalf("reload stock sku: %v", err)
	}
	if reloaded.ManualStockTotal != 8 || reloaded.ManualStockLocked != 0 || reloaded.ManualStockSold != 2 {
		t.Fatalf("unexpected stock lifecycle result: %#v", reloaded)
	}

	unlimited := &productdomain.ProductSKU{
		ProductID:        1,
		SKUCode:          "STOCK-UNLIMITED",
		PriceAmount:      money.FromDecimal(decimal.NewFromInt(100)),
		ManualStockTotal: constants.ManualStockUnlimited,
		IsActive:         true,
	}
	if err := repo.Create(unlimited); err != nil {
		t.Fatalf("create unlimited sku: %v", err)
	}
	if affected, err := repo.ReserveManualStock(unlimited.ID, 1); err != nil || affected != 0 {
		t.Fatalf("unlimited reserve should be no-op, affected=%d err=%v", affected, err)
	}
	if affected, err := repo.ConsumeManualStock(unlimited.ID, 1); err != nil || affected != 0 {
		t.Fatalf("unlimited consume should be no-op, affected=%d err=%v", affected, err)
	}
}

func TestSKUStoreDeleteByProductHidesRowsAndRejectsStockMutations(t *testing.T) {
	repo, db := setupSKUStoreTest(t)
	sku := &productdomain.ProductSKU{
		ProductID:         7,
		SKUCode:           "SOFT-DELETED",
		PriceAmount:       money.FromDecimal(decimal.NewFromInt(100)),
		ManualStockTotal:  10,
		ManualStockLocked: 1,
		IsActive:          true,
	}
	if err := repo.Create(sku); err != nil {
		t.Fatalf("create sku: %v", err)
	}
	if err := repo.DeleteByProduct(sku.ProductID); err != nil {
		t.Fatalf("soft delete skus by product: %v", err)
	}

	var persisted productdomain.ProductSKU
	if err := db.Where("id = ?", sku.ID).First(&persisted).Error; err != nil {
		t.Fatalf("load persisted soft-deleted sku: %v", err)
	}
	if persisted.DeletedAt == nil {
		t.Fatal("expected deleted_at to be persisted")
	}

	byID, err := repo.GetByID(sku.ID)
	if err != nil || byID != nil {
		t.Fatalf("soft-deleted sku must be hidden by id, sku=%#v err=%v", byID, err)
	}
	byCode, err := repo.GetByProductAndCode(sku.ProductID, sku.SKUCode)
	if err != nil || byCode != nil {
		t.Fatalf("soft-deleted sku must be hidden by code, sku=%#v err=%v", byCode, err)
	}
	rows, err := repo.ListByProduct(sku.ProductID, false)
	if err != nil || len(rows) != 0 {
		t.Fatalf("soft-deleted sku must be hidden from list, rows=%#v err=%v", rows, err)
	}

	for operation, mutate := range map[string]func() (int64, error){
		"reserve": func() (int64, error) { return repo.ReserveManualStock(sku.ID, 1) },
		"release": func() (int64, error) { return repo.ReleaseManualStock(sku.ID, 1) },
		"consume": func() (int64, error) { return repo.ConsumeManualStock(sku.ID, 1) },
	} {
		affected, err := mutate()
		if err != nil || affected != 0 {
			t.Fatalf("%s must not mutate soft-deleted sku, affected=%d err=%v", operation, affected, err)
		}
	}
}

func testSKUDisplayMetaAdditiveMigration(t *testing.T, db *gorm.DB) {
	t.Helper()
	repo := NewSKUStore(db)
	row := &productdomain.ProductSKU{ProductID: 111, SKUCode: "EXISTING", PriceAmount: money.FromDecimal(decimal.NewFromInt(879)), ManualStockTotal: 12, ManualStockLocked: 2, ManualStockSold: 3, IsActive: true}
	if err := repo.Create(row); err != nil {
		t.Fatal(err)
	}
	// The isolated fixture models an existing installation without this new column.
	if err := db.Migrator().DropColumn(&productdomain.ProductSKU{}, "DisplayMetaJSON"); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&productdomain.ProductSKU{}); err != nil {
		t.Fatal(err)
	}
	upgraded, err := repo.GetByID(row.ID)
	if err != nil {
		t.Fatal(err)
	}
	if upgraded.ID != row.ID || !upgraded.PriceAmount.Decimal.Equal(row.PriceAmount.Decimal) || upgraded.ManualStockTotal != 12 || upgraded.ManualStockLocked != 2 || upgraded.ManualStockSold != 3 {
		t.Fatalf("migration changed existing identity/price/stock: %#v", upgraded)
	}
	upgraded.DisplayMetaJSON, err = productdomain.NormalizeSKUDisplayMeta(map[string]interface{}{"official_price_usd": "100", "description": map[string]interface{}{"zh-CN": "日常编程"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Update(upgraded); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&productdomain.ProductSKU{}); err != nil {
		t.Fatal(err)
	}
	updated, err := repo.GetByID(row.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.DisplayMetaJSON["official_price_usd"] != "100.00" || updated.DisplayMetaJSON["description"].(map[string]interface{})["zh-CN"] != "日常编程" {
		t.Fatalf("metadata roundtrip lost fields: %#v", updated.DisplayMetaJSON)
	}
}

func TestSKUDisplayMetaAdditiveMigrationSQLite(t *testing.T) {
	_, db := setupSKUStoreTest(t)
	testSKUDisplayMetaAdditiveMigration(t, db)
}

func TestSKUDisplayMetaAdditiveMigrationPostgres(t *testing.T) {
	dsn := os.Getenv("NEEDTOKEN_STOREFRONT_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("isolated PostgreSQL test DSN is not configured")
	}
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("nt_sku_display_%d", time.Now().UnixNano())
	if err := admin.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := admin.Exec("DROP SCHEMA " + schema + " CASCADE").Error; err != nil {
			t.Error(err)
		}
		if sqlDB, err := admin.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	db, err := gorm.Open(postgres.Open(parsed.String()), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	if err := db.AutoMigrate(&productdomain.ProductSKU{}); err != nil {
		t.Fatal(err)
	}
	testSKUDisplayMetaAdditiveMigration(t, db)
}
