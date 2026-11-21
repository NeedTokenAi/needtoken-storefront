package application_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dujiao-next/internal/constants"
	categorycontract "github.com/dujiao-next/internal/modules/catalog/category/contract"
	categorydomain "github.com/dujiao-next/internal/modules/catalog/category/domain"
	productcontract "github.com/dujiao-next/internal/modules/catalog/product/contract"
	productdomain "github.com/dujiao-next/internal/modules/catalog/product/domain"
	couponcontract "github.com/dujiao-next/internal/modules/coupon/contract"
	coupondomain "github.com/dujiao-next/internal/modules/coupon/domain"
	"github.com/dujiao-next/internal/modules/opsbridge/application"
	opscontract "github.com/dujiao-next/internal/modules/opsbridge/contract"
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

type fakeCatRepo struct {
	categorycontract.Repository
	items map[string]*categorydomain.Category
}

func (f *fakeCatRepo) List() ([]categorydomain.Category, error) {
	out := []categorydomain.Category{}
	for _, c := range f.items {
		out = append(out, *c)
	}
	return out, nil
}
func (f *fakeCatRepo) GetByID(id string) (*categorydomain.Category, error) { return f.items[id], nil }
func (f *fakeCatRepo) Update(c *categorydomain.Category) error {
	f.items["1"] = c
	return nil
}

type fakeProdRepo struct {
	productcontract.Repository
	prod    *productdomain.Product
	created []*productdomain.Product
	quick   map[string]interface{}
}

func (f *fakeProdRepo) List(productcontract.ListFilter) ([]productdomain.Product, int64, error) {
	return []productdomain.Product{*f.prod}, 1, nil
}
func (f *fakeProdRepo) GetAdminByID(string) (*productdomain.Product, error) { return f.prod, nil }
func (f *fakeProdRepo) CountBySlug(slug string, _ *string) (int64, error) {
	if f.prod != nil && f.prod.Slug == slug {
		return 1, nil
	}
	for _, p := range f.created {
		if p.Slug == slug {
			return 1, nil
		}
	}
	return 0, nil
}
func (f *fakeProdRepo) Create(item *productdomain.Product) error {
	item.ID = uint(100 + len(f.created) + 1)
	f.created = append(f.created, item)
	return nil
}
func (f *fakeProdRepo) Update(item *productdomain.Product) error {
	f.prod = item
	return nil
}
func (f *fakeProdRepo) QuickUpdate(_ string, fields map[string]interface{}) error {
	f.quick = fields
	return nil
}

type fakeSKURepo struct {
	productcontract.SKURepository
	sku     *productdomain.ProductSKU
	created []*productdomain.ProductSKU
}

func (f *fakeSKURepo) ListByProduct(uint, bool) ([]productdomain.ProductSKU, error) {
	return []productdomain.ProductSKU{*f.sku}, nil
}
func (f *fakeSKURepo) GetByID(uint) (*productdomain.ProductSKU, error) { return f.sku, nil }
func (f *fakeSKURepo) GetByProductAndCode(_ uint, code string) (*productdomain.ProductSKU, error) {
	if f.sku != nil && f.sku.SKUCode == code {
		return f.sku, nil
	}
	for _, s := range f.created {
		if s.SKUCode == code {
			return s, nil
		}
	}
	return nil, nil
}
func (f *fakeSKURepo) Create(s *productdomain.ProductSKU) error {
	s.ID = uint(200 + len(f.created) + 1)
	f.created = append(f.created, s)
	return nil
}
func (f *fakeSKURepo) Update(s *productdomain.ProductSKU) error {
	f.sku = s
	return nil
}

type fakePricingStore struct {
	siteCurrency    string
	displayCurrency string
	usdCnyRate      string
	connRate        float64
	connMarkup      float64
}

func (f *fakePricingStore) GetSitePricing() (string, string, string, string, error) {
	return f.siteCurrency, f.displayCurrency, f.usdCnyRate, "2026-11-21T12:00:00Z", nil
}
func (f *fakePricingStore) UpdateSitePricing(site, display, rate *string) error {
	if site != nil {
		f.siteCurrency = *site
	}
	if display != nil {
		f.displayCurrency = *display
	}
	if rate != nil {
		f.usdCnyRate = *rate
	}
	return nil
}
func (f *fakePricingStore) ListConnectionPricing() ([]opscontract.ConnectionPricing, error) {
	return []opscontract.ConnectionPricing{{
		ID: 1, Name: "dujiao_next", Protocol: "dujiao-next", Status: "active",
		ExchangeRate: "7.25", PriceMarkupPercent: "10", PriceRoundingMode: "none", AutoSyncPrice: true,
	}}, nil
}
func (f *fakePricingStore) UpdateConnectionPricing(_ uint, rate, markup *float64, _ *string, _ *bool) error {
	if rate != nil {
		f.connRate = *rate
	}
	if markup != nil {
		f.connMarkup = *markup
	}
	return nil
}

type fakeCouponRepo struct {
	couponcontract.Repository
	items map[string]*coupondomain.Coupon
	byID  map[uint]*coupondomain.Coupon
}

func (f *fakeCouponRepo) List(couponcontract.ListFilter) ([]coupondomain.Coupon, int64, error) {
	out := []coupondomain.Coupon{}
	for _, c := range f.items {
		out = append(out, *c)
	}
	return out, int64(len(out)), nil
}
func (f *fakeCouponRepo) GetByCode(code string) (*coupondomain.Coupon, error) {
	return f.items[code], nil
}
func (f *fakeCouponRepo) GetByID(id uint) (*coupondomain.Coupon, error) { return f.byID[id], nil }
func (f *fakeCouponRepo) Create(c *coupondomain.Coupon) error {
	c.ID = uint(len(f.items) + 1)
	f.items[c.Code] = c
	f.byID[c.ID] = c
	return nil
}
func (f *fakeCouponRepo) Update(c *coupondomain.Coupon) error {
	f.items[c.Code] = c
	f.byID[c.ID] = c
	return nil
}

func TestOpsBridgeCatalogAndCouponManagement(t *testing.T) {
	cats := &fakeCatRepo{items: map[string]*categorydomain.Category{
		"1": {ID: 1, Slug: "api-plans", NameJSON: jsonmap.JSON{"zh-CN": "API 套餐"}, SortOrder: 10, IsActive: true},
	}}
	prods := &fakeProdRepo{prod: &productdomain.Product{
		ID: 10, CategoryID: 1, Slug: "chatgpt-plus", TitleJSON: jsonmap.JSON{"zh-CN": "ChatGPT Plus"},
		PriceAmount: money.FromDecimal(decimal.NewFromInt(20)), IsActive: true, SortOrder: 5,
	}}
	skus := &fakeSKURepo{sku: &productdomain.ProductSKU{
		ID: 101, ProductID: 10, SKUCode: "MONTHLY", PriceAmount: money.FromDecimal(decimal.NewFromInt(20)), ManualStockTotal: 5, IsActive: true,
	}}
	coupons := &fakeCouponRepo{items: map[string]*coupondomain.Coupon{}, byID: map[uint]*coupondomain.Coupon{}}
	svc := application.NewWithCatalog(orders{}, payments{}, procurements{}, cats, prods, skus, coupons)

	catRes, err := svc.Categories()
	if err != nil || len(catRes.Categories) != 1 || catRes.Categories[0].Name["zh-CN"] != "API 套餐" {
		t.Fatalf("Categories failed: %+v err=%v", catRes, err)
	}
	sortVal := 99
	inactive := false
	updatedCat, err := svc.UpdateCategory("1", opscontract.UpdateCategoryInput{SortOrder: &sortVal, IsActive: &inactive})
	if err != nil || updatedCat.SortOrder != 99 || updatedCat.IsActive {
		t.Fatalf("UpdateCategory failed: %+v err=%v", updatedCat, err)
	}

	prodRes, err := svc.Products(1, 20, "", "")
	if err != nil || len(prodRes.Products) != 1 || len(prodRes.Products[0].SKUs) != 1 {
		t.Fatalf("Products failed: %+v err=%v", prodRes, err)
	}
	if _, err := svc.UpdateProduct("10", opscontract.UpdateProductInput{IsActive: &inactive}); err != nil || prods.quick["is_active"] != false {
		t.Fatalf("UpdateProduct failed: err=%v quick=%+v", err, prods.quick)
	}

	newPrice := "24.90"
	newStock := 18
	updatedSKU, err := svc.UpdateSKU(101, opscontract.UpdateSKUInput{PriceAmount: &newPrice, ManualStockTotal: &newStock})
	if err != nil || updatedSKU.PriceAmount != "24.90" || updatedSKU.ManualStockTotal != 18 {
		t.Fatalf("UpdateSKU failed: %+v err=%v", updatedSKU, err)
	}

	created, err := svc.CreateCoupon(opscontract.CreateCouponInput{
		Code: "save20", Type: constants.CouponTypePercent, Value: "20", UsageLimit: 50, ScopeRefIDs: []uint{10},
	})
	if err != nil || created.Code != "SAVE20" || created.Value != "20.00" || !created.IsActive {
		t.Fatalf("CreateCoupon failed: %+v err=%v", created, err)
	}
	if _, err := svc.CreateCoupon(opscontract.CreateCouponInput{Code: "SAVE20", Type: constants.CouponTypePercent, Value: "10"}); !errors.Is(err, opscontract.ErrConflict) {
		t.Fatalf("duplicate coupon code must return ErrConflict, got %v", err)
	}
	disabled, err := svc.UpdateCouponStatus(created.ID, false)
	if err != nil || disabled.IsActive {
		t.Fatalf("UpdateCouponStatus failed: %+v err=%v", disabled, err)
	}
	listed, err := svc.Coupons(1, 20, "", nil)
	if err != nil || listed.Total != 1 {
		t.Fatalf("Coupons list failed: %+v err=%v", listed, err)
	}

	stockVal := 25
	createdProd, err := svc.CreateProduct(opscontract.CreateProductInput{
		CategoryID:       1,
		Slug:             "claude-pro-1m",
		Title:            map[string]string{"zh-CN": "Claude Pro 月卡", "en-US": "Claude Pro 1M"},
		PriceAmount:      "29.90",
		CostPriceAmount:  "20.00",
		FulfillmentType:  constants.FulfillmentTypeManual,
		ManualStockTotal: &stockVal,
		SKUCode:          "CLAUDE-PRO-1M",
	})
	if err != nil || createdProd.ID == 0 || createdProd.Slug != "claude-pro-1m" || createdProd.PriceAmount != "29.90" || len(createdProd.SKUs) != 1 || createdProd.SKUs[0].SKUCode != "CLAUDE-PRO-1M" {
		t.Fatalf("CreateProduct failed: %+v err=%v", createdProd, err)
	}
	if _, err := svc.CreateProduct(opscontract.CreateProductInput{
		CategoryID: 1, Slug: "claude-pro-1m", Title: map[string]string{"zh-CN": "重复"}, PriceAmount: "10",
	}); !errors.Is(err, opscontract.ErrConflict) {
		t.Fatalf("duplicate product slug must return ErrConflict, got %v", err)
	}
	if _, err := svc.CreateProduct(opscontract.CreateProductInput{
		CategoryID: 999, Slug: "bad-cat", Title: map[string]string{"zh-CN": "坏分类"}, PriceAmount: "10",
	}); !errors.Is(err, opscontract.ErrInvalidInput) {
		t.Fatalf("missing category must return ErrInvalidInput, got %v", err)
	}

	// Full product edit + SKU create/full edit + pricing settings
	newSlug := "chatgpt-plus-v2"
	newProdPrice := "145.00"
	newProdCost := "100.00"
	fullUpdated, err := svc.UpdateProduct("10", opscontract.UpdateProductInput{
		Slug:            &newSlug,
		Title:           map[string]string{"zh-CN": "ChatGPT Plus 升级版", "en-US": "ChatGPT Plus V2"},
		Description:     map[string]string{"zh-CN": "官方直充"},
		PriceAmount:     &newProdPrice,
		CostPriceAmount: &newProdCost,
	})
	if err != nil || fullUpdated.Slug != "chatgpt-plus-v2" || fullUpdated.PriceAmount != "145.00" || fullUpdated.CostPriceAmount != "100.00" || fullUpdated.Description["zh-CN"] != "官方直充" {
		t.Fatalf("full UpdateProduct failed: %+v err=%v", fullUpdated, err)
	}

	newSKUCode := "QUARTERLY"
	newSKUCost := "18.50"
	newSKUSort := 5
	editedSKU, err := svc.UpdateSKU(101, opscontract.UpdateSKUInput{SKUCode: &newSKUCode, CostPriceAmount: &newSKUCost, SortOrder: &newSKUSort})
	if err != nil || editedSKU.SKUCode != "QUARTERLY" || editedSKU.CostPriceAmount != "18.50" || editedSKU.SortOrder != 5 {
		t.Fatalf("full UpdateSKU failed: %+v err=%v", editedSKU, err)
	}

	addedSKU, err := svc.CreateSKU(10, opscontract.CreateSKUInput{SKUCode: "yearly", PriceAmount: "199.00", CostPriceAmount: "120.00"})
	if err != nil || addedSKU.SKUCode != "YEARLY" || addedSKU.PriceAmount != "199.00" {
		t.Fatalf("CreateSKU failed: %+v err=%v", addedSKU, err)
	}

	pricingStore := &fakePricingStore{siteCurrency: "CNY", displayCurrency: "CNY", usdCnyRate: "7.20"}
	svc.WithPricing(pricingStore)
	newDisplay := "USD"
	newRate := "7.30"
	connID := uint(1)
	connRate := 7.35
	connMarkup := 15.0
	pricingRes, err := svc.UpdatePricing(opscontract.UpdatePricingInput{
		DefaultDisplayCurrency: &newDisplay,
		USDCNYRate:             &newRate,
		ConnectionID:           &connID,
		ExchangeRate:           &connRate,
		PriceMarkupPercent:     &connMarkup,
	})
	if err != nil || pricingRes.DefaultDisplayCurrency != "USD" || pricingRes.USDCNYRate != "7.30" || pricingStore.connRate != 7.35 || pricingStore.connMarkup != 15.0 {
		t.Fatalf("UpdatePricing failed: %+v store=%+v err=%v", pricingRes, pricingStore, err)
	}
}
