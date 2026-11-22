package application_test

import (
	"encoding/json"
	"errors"
	"strconv"
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
	giftcardcontract "github.com/dujiao-next/internal/modules/giftcard/contract"
	giftcarddomain "github.com/dujiao-next/internal/modules/giftcard/domain"
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
func (f *fakeCatRepo) CountChildren(categoryID string) (int64, error) {
	var count int64
	for _, c := range f.items {
		if c != nil && c.ParentID > 0 && strconv.FormatUint(uint64(c.ParentID), 10) == categoryID {
			count++
		}
	}
	return count, nil
}
func (f *fakeCatRepo) Update(c *categorydomain.Category) error {
	f.items[strconv.FormatUint(uint64(c.ID), 10)] = c
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

	// Gift card generation & listing through contract repository
	gcRepo := &fakeGiftCardRepo{}
	svc.WithGiftCards(gcRepo)

	ov, err := svc.Overview()
	if err != nil {
		t.Fatalf("Overview failed: %v", err)
	}
	hasGiftRead := false
	hasGiftWrite := false
	for _, capName := range ov.Capabilities {
		if capName == "gift_cards.read" {
			hasGiftRead = true
		}
		if capName == "gift_cards.write" {
			hasGiftWrite = true
		}
	}
	if !hasGiftRead || !hasGiftWrite {
		t.Fatalf("expected gift_cards capabilities in overview, got %v", ov.Capabilities)
	}

	exp := time.Now().UTC().Add(48 * time.Hour).Format(time.RFC3339)
	genRes, err := svc.GenerateGiftCards(opscontract.GenerateGiftCardsInput{
		Name:      "Ops 活动储值卡",
		Quantity:  3,
		Amount:    "50.00",
		ExpiresAt: exp,
	})
	if err != nil || genRes.Created != 3 || len(genRes.Cards) != 3 || !strings.HasPrefix(genRes.BatchNo, "GCB") || genRes.Batch.BatchNo != genRes.BatchNo {
		t.Fatalf("GenerateGiftCards failed: %+v err=%v", genRes, err)
	}
	for _, card := range genRes.Cards {
		if !strings.HasPrefix(card.Code, "GC") || card.Amount != "50.00" || card.Currency != "CNY" || card.Status != giftcarddomain.GiftCardStatusActive || card.BatchNo != genRes.BatchNo {
			t.Fatalf("unexpected generated card: %+v", card)
		}
	}
	if _, err := svc.GenerateGiftCards(opscontract.GenerateGiftCardsInput{Name: "", Quantity: 1, Amount: "10"}); !errors.Is(err, opscontract.ErrInvalidInput) {
		t.Fatalf("empty gift card name must fail with ErrInvalidInput, got %v", err)
	}
	if _, err := svc.GenerateGiftCards(opscontract.GenerateGiftCardsInput{Name: "Bad", Quantity: 1001, Amount: "10"}); !errors.Is(err, opscontract.ErrInvalidInput) {
		t.Fatalf("excessive gift card quantity must fail with ErrInvalidInput, got %v", err)
	}

	listRes, err := svc.GiftCards(1, 20, "", "active", genRes.BatchNo, 0)
	if err != nil || listRes.Total != 3 || len(listRes.GiftCards) != 3 || len(listRes.Cards) != 3 {
		t.Fatalf("GiftCards list failed: %+v err=%v", listRes, err)
	}
}

type fakeGiftCardRepo struct {
	giftcardcontract.Repository
	batches []giftcarddomain.GiftCardBatch
	cards   []giftcarddomain.GiftCard
	nextID  uint
}

func (f *fakeGiftCardRepo) WithinTransaction(fn func(repo giftcardcontract.Repository) error) error {
	return fn(f)
}

func (f *fakeGiftCardRepo) CreateBatch(batch *giftcarddomain.GiftCardBatch, cards []giftcarddomain.GiftCard) error {
	f.nextID++
	batch.ID = f.nextID
	bCopy := *batch
	f.batches = append(f.batches, bCopy)
	for i := range cards {
		f.nextID++
		cards[i].ID = f.nextID
		cards[i].BatchID = &batch.ID
		cCopy := cards[i]
		cCopy.Batch = &bCopy
		f.cards = append(f.cards, cCopy)
	}
	return nil
}

func (f *fakeGiftCardRepo) List(filter giftcardcontract.ListFilter) ([]giftcarddomain.GiftCard, int64, error) {
	var out []giftcarddomain.GiftCard
	for _, c := range f.cards {
		if filter.Status != "" && c.Status != filter.Status {
			continue
		}
		if filter.BatchNo != "" && (c.Batch == nil || !strings.Contains(c.Batch.BatchNo, filter.BatchNo)) {
			continue
		}
		if filter.Code != "" && !strings.Contains(c.Code, filter.Code) {
			continue
		}
		out = append(out, c)
	}
	return out, int64(len(out)), nil
}

type fakeUpstreamStore struct {
	conns []opscontract.UpstreamConnection
}

func (f *fakeUpstreamStore) ListConnections() ([]opscontract.UpstreamConnection, int64, error) {
	return f.conns, int64(len(f.conns)), nil
}

func (f *fakeUpstreamStore) SaveConnection(input opscontract.SaveUpstreamConnectionInput) (*opscontract.UpstreamConnection, error) {
	conn := opscontract.UpstreamConnection{
		ID:       1,
		Name:     input.Name,
		BaseURL:  input.BaseURL,
		ApiKey:   input.ApiKey,
		Protocol: "dujiao-next",
		Status:   "active",
	}
	f.conns = []opscontract.UpstreamConnection{conn}
	return &conn, nil
}

func (f *fakeUpstreamStore) FetchCatalog(connectionID uint) (*opscontract.UpstreamCatalogResult, error) {
	return &opscontract.UpstreamCatalogResult{
		Source:              "storefront",
		ConnectionID:        connectionID,
		CategoriesSupported: true,
		Products: []opscontract.UpstreamCatalogProduct{
			{ID: 1000001, Slug: "tokens-10", IsActive: true, EntitlementContractHash: strings.Repeat("c", 64)},
		},
		Total: 1,
	}, nil
}

func (f *fakeUpstreamStore) ImportOrSync(connectionID uint, input opscontract.ImportUpstreamProductsInput) (*opscontract.ImportUpstreamProductsResult, error) {
	return &opscontract.ImportUpstreamProductsResult{
		Source:        "storefront",
		ConnectionID:  connectionID,
		Requested:     len(input.UpstreamProductIDs),
		ImportedCount: len(input.UpstreamProductIDs),
	}, nil
}

func TestUpstreamBridgeSaveFetchAndImport(t *testing.T) {
	svc := application.New(nil, nil, nil).WithUpstream(&fakeUpstreamStore{})

	if _, err := svc.SaveUpstreamConnection(opscontract.SaveUpstreamConnectionInput{Name: ""}); err != opscontract.ErrInvalidInput {
		t.Fatalf("expected invalid input on empty connection, got %v", err)
	}
	saved, err := svc.SaveUpstreamConnection(opscontract.SaveUpstreamConnectionInput{
		Name:      "NeedToken Glue",
		BaseURL:   "https://glue.needtoken.ai",
		ApiKey:    "glue_key",
		ApiSecret: "glue_secret_32_chars_minimum_test",
	})
	if err != nil || saved.ID != 1 {
		t.Fatalf("SaveUpstreamConnection failed: %v %+v", err, saved)
	}

	cat, err := svc.UpstreamCatalog(saved.ID)
	if err != nil || len(cat.Products) != 1 || cat.Products[0].EntitlementContractHash == "" {
		t.Fatalf("UpstreamCatalog failed: %v %+v", err, cat)
	}

	imp, err := svc.ImportUpstreamProducts(saved.ID, opscontract.ImportUpstreamProductsInput{UpstreamProductIDs: []uint{1000001}})
	if err != nil || imp.ImportedCount != 1 {
		t.Fatalf("ImportUpstreamProducts failed: %v %+v", err, imp)
	}
}

type fakeOrderRefundStore struct {
	records map[string]*opscontract.RefundOrderResult
}

func (f *fakeOrderRefundStore) RefundOrder(orderNo string, input opscontract.RefundOrderInput) (*opscontract.RefundOrderResult, error) {
	if f.records == nil {
		f.records = map[string]*opscontract.RefundOrderResult{}
	}
	if existing, ok := f.records[input.OperationID]; ok {
		if existing.OrderNo != orderNo || existing.RefundedAmount != input.Amount {
			return nil, opscontract.ErrConflict
		}
		return existing, nil
	}
	res := &opscontract.RefundOrderResult{
		Source:           "storefront",
		OrderID:          1,
		OrderNo:          orderNo,
		Status:           constants.OrderStatusRefunded,
		TotalAmount:      input.Amount,
		RefundedAmount:   input.Amount,
		Currency:         "CNY",
		RefundRecordID:   uint(len(f.records) + 1),
		RefundType:       input.Mode,
		SettlementStatus: "settled",
		OperationID:      input.OperationID,
	}
	f.records[input.OperationID] = res
	return res, nil
}

func TestRefundOrderBridgeValidationAndIdempotency(t *testing.T) {
	svc := application.New(orders{}, payments{}, procurements{}).WithOrderRefunds(&fakeOrderRefundStore{})

	ov, err := svc.Overview()
	if err != nil {
		t.Fatalf("Overview failed: %v", err)
	}
	hasRefundWrite := false
	for _, capName := range ov.Capabilities {
		if capName == "refunds.write" {
			hasRefundWrite = true
		}
	}
	if !hasRefundWrite {
		t.Fatalf("expected refunds.write capability, got %v", ov.Capabilities)
	}

	if _, err := svc.RefundOrder("", opscontract.RefundOrderInput{OperationID: "op_1", Amount: "10.00"}); !errors.Is(err, opscontract.ErrInvalidInput) {
		t.Fatalf("empty orderNo must return ErrInvalidInput, got %v", err)
	}
	if _, err := svc.RefundOrder("ORD-1", opscontract.RefundOrderInput{OperationID: "op_1", Mode: "unknown_mode", Amount: "10.00"}); !errors.Is(err, opscontract.ErrInvalidInput) {
		t.Fatalf("invalid mode must return ErrInvalidInput, got %v", err)
	}

	res1, err := svc.RefundOrder("ORD-MOIN-1", opscontract.RefundOrderInput{
		OperationID:   "ops_refund_moin_1",
		Amount:        "85",
		Channel:       "moin_cash",
		ExternalTxRef: "moin_rf_001",
	})
	if err != nil || res1.RefundedAmount != "85.00" || res1.SettlementStatus != "settled" || res1.RefundType != "external_original" {
		t.Fatalf("RefundOrder failed: %v %+v", err, res1)
	}

	resDup, err := svc.RefundOrder("ORD-MOIN-1", opscontract.RefundOrderInput{
		OperationID:   "ops_refund_moin_1",
		Amount:        "85.00",
		Channel:       "moin_cash",
		ExternalTxRef: "moin_rf_001",
	})
	if err != nil || resDup.RefundRecordID != res1.RefundRecordID {
		t.Fatalf("idempotent replay failed: %v %+v", err, resDup)
	}

	if _, err := svc.RefundOrder("ORD-MOIN-1", opscontract.RefundOrderInput{
		OperationID: "ops_refund_moin_1",
		Amount:      "90.00",
	}); !errors.Is(err, opscontract.ErrConflict) {
		t.Fatalf("tampered replay must return ErrConflict, got %v", err)
	}
}

type multiSKURepo struct {
	productcontract.SKURepository
	items   map[uint]*productdomain.ProductSKU
	deleted []uint
}

func (m *multiSKURepo) ListByProduct(productID uint, onlyActive bool) ([]productdomain.ProductSKU, error) {
	out := make([]productdomain.ProductSKU, 0, len(m.items))
	for _, sku := range m.items {
		if sku.ProductID != productID {
			continue
		}
		if onlyActive && !sku.IsActive {
			continue
		}
		out = append(out, *sku)
	}
	return out, nil
}
func (m *multiSKURepo) GetByID(id uint) (*productdomain.ProductSKU, error) {
	if s, ok := m.items[id]; ok {
		cp := *s
		return &cp, nil
	}
	return nil, nil
}
func (m *multiSKURepo) GetByProductAndCode(productID uint, code string) (*productdomain.ProductSKU, error) {
	for _, s := range m.items {
		if s.ProductID == productID && s.SKUCode == code {
			cp := *s
			return &cp, nil
		}
	}
	return nil, nil
}
func (m *multiSKURepo) Update(s *productdomain.ProductSKU) error {
	cp := *s
	m.items[s.ID] = &cp
	return nil
}
func (m *multiSKURepo) Delete(id uint) error {
	m.deleted = append(m.deleted, id)
	delete(m.items, id)
	return nil
}

type fakeCardSecretCounter struct {
	total     map[uint]int64
	available map[uint]int64
	used      map[uint]int64
}

func (f *fakeCardSecretCounter) CountByProduct(_ uint, skuID uint) (int64, int64, int64, error) {
	return f.total[skuID], f.available[skuID], f.used[skuID], nil
}

type fakeProductDeleteStore struct {
	errs    map[string]error
	deleted []string
}

func (f *fakeProductDeleteStore) Delete(id string) error {
	if err, ok := f.errs[id]; ok {
		return err
	}
	f.deleted = append(f.deleted, id)
	return nil
}

func TestOpsBridgeBugFixesAndSafeDeleteGuards(t *testing.T) {
	// Parent category #1 has child category #2; products may only be assigned to leaf #2.
	cats := &fakeCatRepo{items: map[string]*categorydomain.Category{
		"1": {ID: 1, ParentID: 0, Slug: "ai-parent", NameJSON: jsonmap.JSON{"zh-CN": "AI 父类"}, IsActive: true},
		"2": {ID: 2, ParentID: 1, Slug: "ai-child", NameJSON: jsonmap.JSON{"zh-CN": "AI 子类"}, IsActive: true},
	}}
	prod := &productdomain.Product{
		ID:              10,
		CategoryID:      2,
		Slug:            "multi-sku-plan",
		TitleJSON:       jsonmap.JSON{"zh-CN": "多规格商品"},
		PriceAmount:     money.FromDecimal(decimal.NewFromInt(10)),
		CostPriceAmount: money.FromDecimal(decimal.NewFromInt(5)),
		FulfillmentType: constants.FulfillmentTypeAuto,
		WholesalePrices: productdomain.WholesalePriceTiers{
			{SKUID: 101, SKUCode: "BASIC", MinQuantity: 5, UnitPrice: money.FromDecimal(decimal.NewFromInt(8))},
			{SKUID: 102, SKUCode: "PRO", MinQuantity: 5, UnitPrice: money.FromDecimal(decimal.NewFromInt(40))},
		},
		IsActive: true,
	}
	prods := &fakeProdRepo{prod: prod}
	skus := &multiSKURepo{items: map[uint]*productdomain.ProductSKU{
		101: {ID: 101, ProductID: 10, SKUCode: "BASIC", PriceAmount: money.FromDecimal(decimal.NewFromInt(10)), CostPriceAmount: money.FromDecimal(decimal.NewFromInt(5)), IsActive: true},
		102: {ID: 102, ProductID: 10, SKUCode: "PRO", PriceAmount: money.FromDecimal(decimal.NewFromInt(50)), CostPriceAmount: money.FromDecimal(decimal.NewFromInt(30)), IsActive: true},
	}}
	secrets := &fakeCardSecretCounter{
		total:     map[uint]int64{101: 2, 102: 3},
		available: map[uint]int64{101: 1, 102: 0},
		used:      map[uint]int64{101: 1, 102: 3},
	}
	delStore := &fakeProductDeleteStore{
		errs: map[string]error{
			"20": opscontract.ErrProductHasStock,
			"21": opscontract.ErrProductHasOrderRecord,
			"22": opscontract.ErrNotFound,
		},
	}
	svc := application.NewWithCatalog(orders{}, payments{}, procurements{}, cats, prods, skus, nil).
		WithCardSecretStock(secrets).
		WithProductDeleteStore(delStore)

	// Bug fix #1: reject assigning product to a parent category that has children.
	if _, err := svc.CreateProduct(opscontract.CreateProductInput{
		CategoryID: 1, Slug: "blocked-parent", Title: map[string]string{"zh-CN": "父分类禁止"}, PriceAmount: "10.00",
	}); !errors.Is(err, opscontract.ErrInvalidInput) {
		t.Fatalf("CreateProduct on parent category with children must fail with ErrInvalidInput, got %v", err)
	}
	parentID := uint(1)
	if _, err := svc.UpdateProduct("10", opscontract.UpdateProductInput{CategoryID: &parentID}); !errors.Is(err, opscontract.ErrInvalidInput) {
		t.Fatalf("UpdateProduct moving to parent category with children must fail with ErrInvalidInput, got %v", err)
	}

	// Bug fix #2: UpdateSKU recalculates product display price from lowest active SKU, not current SKU.
	higherPrice := "60.00"
	if _, err := svc.UpdateSKU(102, opscontract.UpdateSKUInput{PriceAmount: &higherPrice}); err != nil {
		t.Fatalf("UpdateSKU 102 failed: %v", err)
	}
	if got := prods.quick["price_amount"].(money.Amount).String(); got != "10.00" {
		t.Fatalf("product price_amount must stay lowest active SKU (10.00), got %s", got)
	}
	// Deactivating the cheaper SKU must promote the remaining active SKU price (60.00).
	inactive := false
	if _, err := svc.UpdateSKU(101, opscontract.UpdateSKUInput{IsActive: &inactive}); err != nil {
		t.Fatalf("deactivate SKU 101 failed: %v", err)
	}
	if got := prods.quick["price_amount"].(money.Amount).String(); got != "60.00" {
		t.Fatalf("deactivating cheaper SKU must recalculate product price_amount to 60.00, got %s", got)
	}
	// Re-activate 101 so both are active again.
	active := true
	if _, err := svc.UpdateSKU(101, opscontract.UpdateSKUInput{IsActive: &active}); err != nil {
		t.Fatalf("reactivate SKU 101 failed: %v", err)
	}

	// DeleteSKU guard: auto-fulfillment SKU 101 still has available card secrets.
	if err := svc.DeleteSKU(101); !errors.Is(err, opscontract.ErrSKUHasCardSecretStock) {
		t.Fatalf("DeleteSKU with available secrets must return ErrSKUHasCardSecretStock, got %v", err)
	}

	// DeleteSKU succeeds for SKU 102 (all secrets used) and prunes dangling wholesale tier for 102.
	if err := svc.DeleteSKU(102); err != nil {
		t.Fatalf("DeleteSKU 102 should succeed, got %v", err)
	}
	if len(prods.prod.WholesalePrices) != 1 || prods.prod.WholesalePrices[0].SKUID != 101 {
		t.Fatalf("dangling wholesale price for deleted SKU 102 must be pruned: %+v", prods.prod.WholesalePrices)
	}
	if prods.prod.PriceAmount.String() != "10.00" || prods.prod.CostPriceAmount.String() != "5.00" {
		t.Fatalf("product price/cost should recalculate from remaining active SKU: price=%s cost=%s", prods.prod.PriceAmount.String(), prods.prod.CostPriceAmount.String())
	}

	// DeleteSKU guard: cannot delete the last remaining active SKU.
	secrets.available[101] = 0
	secrets.used[101] = 2
	if err := svc.DeleteSKU(101); !errors.Is(err, opscontract.ErrProductRequiresActiveSKU) {
		t.Fatalf("DeleteSKU removing last active SKU must return ErrProductRequiresActiveSKU, got %v", err)
	}

	// BatchDeleteProducts validates bounds (1..100) and reports per-item error codes.
	if _, err := svc.BatchDeleteProducts(opscontract.BatchDeleteProductsInput{}); !errors.Is(err, opscontract.ErrInvalidInput) {
		t.Fatalf("empty batch delete must fail with ErrInvalidInput, got %v", err)
	}
	tooMany := make([]uint, 101)
	if _, err := svc.BatchDeleteProducts(opscontract.BatchDeleteProductsInput{IDs: tooMany}); !errors.Is(err, opscontract.ErrInvalidInput) {
		t.Fatalf(">100 batch delete must fail with ErrInvalidInput, got %v", err)
	}
	batchRes, err := svc.BatchDeleteProducts(opscontract.BatchDeleteProductsInput{IDs: []uint{19, 20, 21, 22}})
	if err != nil {
		t.Fatalf("BatchDeleteProducts failed: %v", err)
	}
	if batchRes.Total != 4 || batchRes.Succeeded != 1 || batchRes.Failed != 3 || len(batchRes.Errors) != 3 {
		t.Fatalf("unexpected batch delete summary: %+v", batchRes)
	}
	if batchRes.Errors[0].Error != "product_has_stock" || batchRes.Errors[1].Error != "product_has_order_record" || batchRes.Errors[2].Error != "not_found" {
		t.Fatalf("unexpected batch delete error codes: %+v", batchRes.Errors)
	}
}
