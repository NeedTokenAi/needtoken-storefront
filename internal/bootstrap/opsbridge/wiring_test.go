package opsbridge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dujiao-next/internal/app/container"
	catalogproductbootstrap "github.com/dujiao-next/internal/bootstrap/catalogproduct"
	"github.com/dujiao-next/internal/constants"
	cardsecretdomain "github.com/dujiao-next/internal/modules/cardsecret/domain"
	cardsecretgormstore "github.com/dujiao-next/internal/modules/cardsecret/infrastructure/gormstore"
	cartdomain "github.com/dujiao-next/internal/modules/cart/domain"
	cartgormstore "github.com/dujiao-next/internal/modules/cart/infrastructure/gormstore"
	categoryapp "github.com/dujiao-next/internal/modules/catalog/category/application"
	categorydomain "github.com/dujiao-next/internal/modules/catalog/category/domain"
	categorygormstore "github.com/dujiao-next/internal/modules/catalog/category/infrastructure/gormstore"
	mappingdomain "github.com/dujiao-next/internal/modules/catalog/mapping/domain"
	mappinggormstore "github.com/dujiao-next/internal/modules/catalog/mapping/infrastructure/gormstore"
	productdomain "github.com/dujiao-next/internal/modules/catalog/product/domain"
	productgormstore "github.com/dujiao-next/internal/modules/catalog/product/store/gormstore"
	memberleveldomain "github.com/dujiao-next/internal/modules/memberlevel/domain"
	memberlevelgormstore "github.com/dujiao-next/internal/modules/memberlevel/infrastructure/gormstore"
	opscontract "github.com/dujiao-next/internal/modules/opsbridge/contract"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	ordergormstore "github.com/dujiao-next/internal/modules/order/infrastructure/gormstore"
	paymentdomain "github.com/dujiao-next/internal/modules/payment/domain"
	paymentgormstore "github.com/dujiao-next/internal/modules/payment/infrastructure/gormstore"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/dujiao-next/internal/shared/money"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func setupOpsBridgeTestRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dsn := fmt.Sprintf("file:opsbridge_catalog_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql.DB: %v", err)
	}
	// Mirror production SQLite MaxOpenConns=1 to catch any transaction self-deadlock.
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(
		&categorydomain.Category{},
		&productdomain.Product{},
		&productdomain.ProductSKU{},
		&cardsecretdomain.Secret{},
		&cardsecretdomain.Batch{},
		&memberleveldomain.MemberLevelPrice{},
		&memberleveldomain.UserProductPrice{},
		&cartdomain.Item{},
		&mappingdomain.Mapping{},
		&mappingdomain.SKUMapping{},
		&orderdomain.Order{},
		&orderdomain.OrderItem{},
		&paymentdomain.PaymentChannel{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	catRepo := categorygormstore.NewCategoryStore(db)
	prodRepo := productgormstore.NewProductStore(db)
	skuRepo := productgormstore.NewSKUStore(db)
	cardSecretRepo := cardsecretgormstore.New(db)
	orderStore := ordergormstore.New(db, "test-guest-secret-at-least-32-bytes")

	prodServices := catalogproductbootstrap.New(catalogproductbootstrap.Dependencies{
		Products:          prodRepo,
		SKUs:              skuRepo,
		CardSecrets:       cardSecretRepo,
		CardSecretBatches: cardsecretgormstore.NewBatch(db),
		Categories:        catRepo,
		MemberLevelPrices: memberlevelgormstore.NewPriceStore(db),
		UserPrices:        memberlevelgormstore.NewUserPriceStore(db),
		Carts:             cartgormstore.New(db),
		ProductMappings:   mappinggormstore.NewMappingStore(db),
		Orders:            orderStore,
		PaymentChannels:   paymentgormstore.NewChannelStore(db),
	})

	c := &container.Container{
		OrderStore:          orderStore,
		CategoryRepo:        catRepo,
		CategoryService:     categoryapp.NewService(catRepo),
		ProductRepo:         prodRepo,
		ProductSKURepo:      skuRepo,
		ProductAdminService: prodServices.Admin,
		CardSecretRepo:      cardSecretRepo,
	}

	h := NewHandler(c)
	r := gin.New()
	g := r.Group("/api/v1/channel/needtoken-ops")
	g.GET("/catalog/categories", h.Categories)
	g.POST("/catalog/categories", h.CreateCategory)
	g.PUT("/catalog/categories/:id", h.UpdateCategory)
	g.PATCH("/catalog/categories/:id", h.UpdateCategory)
	g.DELETE("/catalog/categories/:id", h.DeleteCategory)
	g.GET("/catalog/products", h.Products)
	g.POST("/catalog/products", h.CreateProduct)
	g.PATCH("/catalog/products/:id", h.UpdateProduct)
	g.DELETE("/catalog/products/:id", h.DeleteProduct)
	g.POST("/catalog/products/batch-delete", h.BatchDeleteProducts)
	g.POST("/catalog/products/:id/skus", h.CreateSKU)
	g.PATCH("/catalog/skus/:id", h.UpdateSKU)
	g.DELETE("/catalog/skus/:id", h.DeleteSKU)

	return r, db
}

func doOpsJSON(t *testing.T, r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode request: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestOpsBridgeCatalogCRUDAndSafeDeletesOnSQLite(t *testing.T) {
	r, db := setupOpsBridgeTestRouter(t)

	// 1. Create root category and second-level child category; reject third-level and missing parent.
	w := doOpsJSON(t, r, http.MethodPost, "/api/v1/channel/needtoken-ops/catalog/categories", map[string]any{
		"parent_id":  0,
		"slug":       "ai-root",
		"name":       map[string]string{"zh-CN": "AI 服务", "en-US": "AI Services"},
		"icon":       "sparkles",
		"sort_order": 10,
		"is_active":  true,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("create root category: status=%d body=%s", w.Code, w.Body.String())
	}
	var rootResp struct {
		StatusCode int                  `json:"status_code"`
		Data       opscontract.Category `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &rootResp)
	rootID := rootResp.Data.ID
	if rootID == 0 || rootResp.Data.Icon != "sparkles" {
		t.Fatalf("unexpected root category: %+v", rootResp.Data)
	}

	w = doOpsJSON(t, r, http.MethodPost, "/api/v1/channel/needtoken-ops/catalog/categories", map[string]any{
		"parent_id":  rootID,
		"slug":       "ai-child",
		"name":       map[string]string{"zh-CN": "API 套餐"},
		"sort_order": 5,
		"is_active":  false,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("create child category: status=%d body=%s", w.Code, w.Body.String())
	}
	var childResp struct {
		Data opscontract.Category `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &childResp)
	childID := childResp.Data.ID
	if childID == 0 || childResp.Data.ParentID != rootID || childResp.Data.IsActive {
		t.Fatalf("unexpected child category: %+v", childResp.Data)
	}

	// Third-level category must fail with 400.
	w = doOpsJSON(t, r, http.MethodPost, "/api/v1/channel/needtoken-ops/catalog/categories", map[string]any{
		"parent_id": childID,
		"slug":      "ai-grandchild",
		"name":      map[string]string{"zh-CN": "三级非法"},
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("third-level category must return 400, got %d: %s", w.Code, w.Body.String())
	}

	// Missing parent must fail with 400.
	w = doOpsJSON(t, r, http.MethodPost, "/api/v1/channel/needtoken-ops/catalog/categories", map[string]any{
		"parent_id": 99999,
		"slug":      "orphan-child",
		"name":      map[string]string{"zh-CN": "孤儿分类"},
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("missing parent category must return 400, got %d: %s", w.Code, w.Body.String())
	}

	// 2. PUT full update + PATCH compatibility on category.
	w = doOpsJSON(t, r, http.MethodPut, fmt.Sprintf("/api/v1/channel/needtoken-ops/catalog/categories/%d", childID), map[string]any{
		"parent_id":  rootID,
		"slug":       "ai-plans",
		"name":       map[string]string{"zh-CN": "API 订阅", "en-US": "API Plans"},
		"icon":       "zap",
		"sort_order": 20,
		"is_active":  true,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("PUT category failed: status=%d body=%s", w.Code, w.Body.String())
	}
	_ = json.Unmarshal(w.Body.Bytes(), &childResp)
	if childResp.Data.Slug != "ai-plans" || childResp.Data.Icon != "zap" || !childResp.Data.IsActive || childResp.Data.SortOrder != 20 {
		t.Fatalf("unexpected PUT category result: %+v", childResp.Data)
	}

	w = doOpsJSON(t, r, http.MethodPatch, fmt.Sprintf("/api/v1/channel/needtoken-ops/catalog/categories/%d", childID), map[string]any{
		"sort_order": 30,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("PATCH category failed: status=%d body=%s", w.Code, w.Body.String())
	}

	// Deleting root category with children must return 409 + category_in_use.
	w = doOpsJSON(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/channel/needtoken-ops/catalog/categories/%d", rootID), nil)
	if w.Code != http.StatusConflict {
		t.Fatalf("delete category with children must return 409, got %d: %s", w.Code, w.Body.String())
	}
	var errResp struct {
		ErrorCode string `json:"error_code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.ErrorCode != "category_in_use" {
		t.Fatalf("expected error_code=category_in_use, got %+v (%s)", errResp, w.Body.String())
	}

	// 3. Bug fix: creating product under parent category (with children) must fail with 400.
	w = doOpsJSON(t, r, http.MethodPost, "/api/v1/channel/needtoken-ops/catalog/products", map[string]any{
		"category_id":      rootID,
		"slug":             "invalid-parent-prod",
		"title":            map[string]string{"zh-CN": "父分类商品"},
		"price_amount":     "20.00",
		"fulfillment_type": constants.FulfillmentTypeAuto,
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("product on parent category with children must fail 400, got %d: %s", w.Code, w.Body.String())
	}

	// Create auto-fulfillment product on leaf child category.
	w = doOpsJSON(t, r, http.MethodPost, "/api/v1/channel/needtoken-ops/catalog/products", map[string]any{
		"category_id":       childID,
		"slug":              "api-pack",
		"title":             map[string]string{"zh-CN": "API 额度包"},
		"price_amount":      "10.00",
		"cost_price_amount": "6.00",
		"fulfillment_type":  constants.FulfillmentTypeAuto,
		"sku_code":          "PACK-10",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("create product: status=%d body=%s", w.Code, w.Body.String())
	}
	var prodResp struct {
		Data opscontract.Product `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &prodResp)
	prodID := prodResp.Data.ID
	sku1ID := prodResp.Data.SKUs[0].ID

	// Add second SKU at higher price (50.00).
	w = doOpsJSON(t, r, http.MethodPost, fmt.Sprintf("/api/v1/channel/needtoken-ops/catalog/products/%d/skus", prodID), map[string]any{
		"sku_code":          "PACK-50",
		"price_amount":      "50.00",
		"cost_price_amount": "32.00",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("create second sku: status=%d body=%s", w.Code, w.Body.String())
	}
	var sku2Resp struct {
		Data opscontract.ProductSKU `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &sku2Resp)
	sku2ID := sku2Resp.Data.ID

	// Bug fix: updating higher SKU to 55.00 must keep product price at lowest active SKU (10.00).
	w = doOpsJSON(t, r, http.MethodPatch, fmt.Sprintf("/api/v1/channel/needtoken-ops/catalog/skus/%d", sku2ID), map[string]any{
		"price_amount": "55.00",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("patch sku2: status=%d body=%s", w.Code, w.Body.String())
	}
	var dbProd productdomain.Product
	if err := db.First(&dbProd, prodID).Error; err != nil {
		t.Fatalf("load product: %v", err)
	}
	if dbProd.PriceAmount.String() != "10.00" {
		t.Fatalf("expected product price_amount 10.00, got %s", dbProd.PriceAmount.String())
	}

	// Attach wholesale prices referencing both SKUs, and add available card secret to sku1.
	dbProd.WholesalePrices = productdomain.WholesalePriceTiers{
		{SKUID: sku1ID, SKUCode: "PACK-10", MinQuantity: 5, UnitPrice: money.FromDecimal(decimal.NewFromInt(8))},
		{SKUID: sku2ID, SKUCode: "PACK-50", MinQuantity: 5, UnitPrice: money.FromDecimal(decimal.NewFromInt(45))},
	}
	if err := db.Save(&dbProd).Error; err != nil {
		t.Fatalf("save wholesale prices: %v", err)
	}
	if err := db.Create(&cardsecretdomain.Secret{
		ProductID: prodID,
		SKUID:     sku1ID,
		Secret:    "CARD-AVAILABLE-1",
		Status:    cardsecretdomain.StatusAvailable,
	}).Error; err != nil {
		t.Fatalf("seed card secret: %v", err)
	}

	// Deleting product with stock must return 409 + product_has_stock.
	w = doOpsJSON(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/channel/needtoken-ops/catalog/products/%d", prodID), nil)
	if w.Code != http.StatusConflict {
		t.Fatalf("delete product with stock must return 409, got %d: %s", w.Code, w.Body.String())
	}
	errResp.ErrorCode = ""
	_ = json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.ErrorCode != "product_has_stock" {
		t.Fatalf("expected product_has_stock, got %q", errResp.ErrorCode)
	}

	// Deleting sku1 with available secret must return 409 + sku_has_card_secret_stock.
	w = doOpsJSON(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/channel/needtoken-ops/catalog/skus/%d", sku1ID), nil)
	if w.Code != http.StatusConflict {
		t.Fatalf("delete sku with available secrets must return 409, got %d: %s", w.Code, w.Body.String())
	}
	errResp.ErrorCode = ""
	_ = json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.ErrorCode != "sku_has_card_secret_stock" {
		t.Fatalf("expected sku_has_card_secret_stock, got %q", errResp.ErrorCode)
	}

	// Hard-delete sku2 (no card secrets): prunes wholesale price for sku2, keeps product price 10.00, allows reusing PACK-50 code.
	w = doOpsJSON(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/channel/needtoken-ops/catalog/skus/%d", sku2ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete sku2 failed: status=%d body=%s", w.Code, w.Body.String())
	}
	var skuCount int64
	db.Unscoped().Model(&productdomain.ProductSKU{}).Where("id = ?", sku2ID).Count(&skuCount)
	if skuCount != 0 {
		t.Fatalf("sku2 must be hard-deleted, unscoped count=%d", skuCount)
	}
	if err := db.First(&dbProd, prodID).Error; err != nil {
		t.Fatalf("reload product: %v", err)
	}
	if len(dbProd.WholesalePrices) != 1 || dbProd.WholesalePrices[0].SKUID != sku1ID {
		t.Fatalf("wholesale tier for deleted sku2 not pruned: %+v", dbProd.WholesalePrices)
	}
	// Recreating PACK-50 succeeds without unique index collision because DeleteSKU hard-deletes.
	w = doOpsJSON(t, r, http.MethodPost, fmt.Sprintf("/api/v1/channel/needtoken-ops/catalog/products/%d/skus", prodID), map[string]any{
		"sku_code":     "PACK-50",
		"price_amount": "48.00",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("recreate PACK-50 after hard delete failed: status=%d body=%s", w.Code, w.Body.String())
	}
	var recreatedSKU struct {
		Data opscontract.ProductSKU `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &recreatedSKU)
	w = doOpsJSON(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/channel/needtoken-ops/catalog/skus/%d", recreatedSKU.Data.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete recreated PACK-50 failed: status=%d body=%s", w.Code, w.Body.String())
	}

	// Mark card secret used and verify deleting last remaining active SKU returns 409 + product_requires_active_sku.
	db.Model(&cardsecretdomain.Secret{}).Where("product_id = ?", prodID).Update("status", cardsecretdomain.StatusUsed)
	w = doOpsJSON(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/channel/needtoken-ops/catalog/skus/%d", sku1ID), nil)
	if w.Code != http.StatusConflict {
		t.Fatalf("delete last active sku must return 409, got %d: %s", w.Code, w.Body.String())
	}
	errResp.ErrorCode = ""
	_ = json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.ErrorCode != "product_requires_active_sku" {
		t.Fatalf("expected product_requires_active_sku, got %q", errResp.ErrorCode)
	}

	// Create a second product with an order record to test product_has_order_record & batch delete.
	w = doOpsJSON(t, r, http.MethodPost, "/api/v1/channel/needtoken-ops/catalog/products", map[string]any{
		"category_id":  childID,
		"slug":         "ordered-prod",
		"title":        map[string]string{"zh-CN": "已成交商品"},
		"price_amount": "15.00",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("create ordered product: status=%d body=%s", w.Code, w.Body.String())
	}
	var orderedProdResp struct {
		Data opscontract.Product `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &orderedProdResp)
	orderedProdID := orderedProdResp.Data.ID
	if err := db.Create(&orderdomain.OrderItem{
		OrderID:   1,
		ProductID: orderedProdID,
		SKUID:     orderedProdResp.Data.SKUs[0].ID,
		TitleJSON: jsonmap.JSON{"zh-CN": "已成交商品"},
		Quantity:  1,
	}).Error; err != nil {
		t.Fatalf("seed order item: %v", err)
	}

	// Batch delete: prodID (now no available/reserved stock -> succeeds), orderedProdID (has order -> fails), 99999 (not found).
	w = doOpsJSON(t, r, http.MethodPost, "/api/v1/channel/needtoken-ops/catalog/products/batch-delete", map[string]any{
		"ids": []uint{prodID, orderedProdID, 99999},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("batch delete: status=%d body=%s", w.Code, w.Body.String())
	}
	var batchResp struct {
		Data opscontract.BatchDeleteResult `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &batchResp)
	if batchResp.Data.Total != 3 || batchResp.Data.Succeeded != 1 || batchResp.Data.Failed != 2 {
		t.Fatalf("unexpected batch delete result: %+v", batchResp.Data)
	}
	if batchResp.Data.Errors[0].Error != "product_has_order_record" || batchResp.Data.Errors[1].Error != "not_found" {
		t.Fatalf("unexpected batch delete errors: %+v", batchResp.Data.Errors)
	}

	// Deleting child category while orderedProdID still references it returns category_in_use; after deleting order item & product, child & root categories delete cleanly.
	w = doOpsJSON(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/channel/needtoken-ops/catalog/categories/%d", childID), nil)
	if w.Code != http.StatusConflict {
		t.Fatalf("delete category with product must return 409, got %d: %s", w.Code, w.Body.String())
	}
	db.Where("product_id = ?", orderedProdID).Delete(&orderdomain.OrderItem{})
	w = doOpsJSON(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/channel/needtoken-ops/catalog/products/%d", orderedProdID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete product after clearing order item failed: status=%d body=%s", w.Code, w.Body.String())
	}
	w = doOpsJSON(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/channel/needtoken-ops/catalog/categories/%d", childID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete empty child category failed: status=%d body=%s", w.Code, w.Body.String())
	}
	w = doOpsJSON(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/channel/needtoken-ops/catalog/categories/%d", rootID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete empty root category failed: status=%d body=%s", w.Code, w.Body.String())
	}
}
