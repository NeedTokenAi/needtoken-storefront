package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	orderdomain "github.com/dujiao-next/internal/modules/order/domain"

	coupondomain "github.com/dujiao-next/internal/modules/coupon/domain"

	promotiondomain "github.com/dujiao-next/internal/modules/promotion/domain"

	memberleveldomain "github.com/dujiao-next/internal/modules/memberlevel/domain"

	productdomain "github.com/dujiao-next/internal/modules/catalog/product/domain"

	categorydomain "github.com/dujiao-next/internal/modules/catalog/category/domain"

	productgormstore "github.com/dujiao-next/internal/modules/catalog/product/store/gormstore"

	userstore "github.com/dujiao-next/internal/modules/identity/user/infrastructure/gormstore"

	userdomain "github.com/dujiao-next/internal/modules/identity/user/domain"

	"github.com/dujiao-next/internal/constants"
	mappingcontract "github.com/dujiao-next/internal/modules/catalog/mapping/contract"
	couponcontract "github.com/dujiao-next/internal/modules/coupon/contract"
	coupongormstore "github.com/dujiao-next/internal/modules/coupon/infrastructure/gormstore"
	memberlevelapp "github.com/dujiao-next/internal/modules/memberlevel/application"
	memberlevelgormstore "github.com/dujiao-next/internal/modules/memberlevel/infrastructure/gormstore"
	promotiongormstore "github.com/dujiao-next/internal/modules/promotion/infrastructure/gormstore"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/dujiao-next/internal/shared/money"

	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type wholesaleOrderFixture struct {
	db      *gorm.DB
	svc     *OrderService
	product productdomain.Product
	sku     productdomain.ProductSKU
	user    userdomain.User
}

func setupWholesaleOrderFixture(t *testing.T, name string, wholesalePrices productdomain.WholesalePriceTiers, promotionPercent *decimal.Decimal, memberRate *decimal.Decimal) wholesaleOrderFixture {
	t.Helper()

	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared", name, time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(
		&categorydomain.Category{},
		&productdomain.Product{},
		&productdomain.ProductSKU{},
		&promotiondomain.Promotion{},
		&coupondomain.Coupon{},
		&coupondomain.CouponUsage{},
		&userdomain.User{},
		&memberleveldomain.MemberLevel{},
		&memberleveldomain.MemberLevelPrice{},
	); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	now := time.Now()
	category := categorydomain.Category{
		Slug:      name + "-category",
		NameJSON:  jsonmap.JSON{"zh-CN": "测试分类"},
		CreatedAt: now,
	}
	if err := db.Create(&category).Error; err != nil {
		t.Fatalf("create category failed: %v", err)
	}

	var user userdomain.User
	if memberRate != nil {
		level := memberleveldomain.MemberLevel{
			NameJSON:     jsonmap.JSON{"zh-CN": "批发会员"},
			Slug:         name + "-level",
			DiscountRate: money.FromDecimal(*memberRate),
			IsActive:     true,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := db.Create(&level).Error; err != nil {
			t.Fatalf("create member level failed: %v", err)
		}
		user = userdomain.User{
			Email:         name + "@example.com",
			PasswordHash:  "hash",
			Status:        "active",
			MemberLevelID: level.ID,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("create user failed: %v", err)
		}
	}

	product := productdomain.Product{
		CategoryID:      category.ID,
		Slug:            name + "-product",
		TitleJSON:       jsonmap.JSON{"zh-CN": "批发测试商品"},
		PriceAmount:     money.FromDecimal(decimal.NewFromInt(100)),
		WholesalePrices: wholesalePrices,
		PurchaseType:    constants.ProductPurchaseMember,
		FulfillmentType: constants.FulfillmentTypeAuto,
		IsActive:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("create product failed: %v", err)
	}

	sku := productdomain.ProductSKU{
		ProductID:   product.ID,
		SKUCode:     productdomain.DefaultSKUCode,
		PriceAmount: money.FromDecimal(decimal.NewFromInt(100)),
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := db.Create(&sku).Error; err != nil {
		t.Fatalf("create sku failed: %v", err)
	}

	if promotionPercent != nil {
		promotion := promotiondomain.Promotion{
			Name:       name + "-promotion",
			ScopeType:  constants.ScopeTypeProduct,
			ScopeRefID: product.ID,
			Type:       constants.PromotionTypePercent,
			Value:      money.FromDecimal(*promotionPercent),
			MinAmount:  money.FromDecimal(decimal.Zero),
			IsActive:   true,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if err := db.Create(&promotion).Error; err != nil {
			t.Fatalf("create promotion failed: %v", err)
		}
	}

	userRepo := userstore.New(db)
	levelRepo := memberlevelgormstore.NewLevelStore(db)
	priceRepo := memberlevelgormstore.NewPriceStore(db)
	svc := NewOrderService(OrderServiceOptions{
		UserStore:          userRepo,
		ProductStore:       productgormstore.NewProductStore(db),
		ProductSKUStore:    productgormstore.NewSKUStore(db),
		PromotionRepo:      promotiongormstore.New(db),
		CouponStore:        coupongormstore.New(db),
		CouponUsageStore:   coupongormstore.NewUsageStore(db),
		MemberLevelService: memberlevelapp.NewService(levelRepo, priceRepo, memberlevelgormstore.NewUserStore(db)),
		ExpireMinutes:      15,
	})

	return wholesaleOrderFixture{db: db, svc: svc, product: product, sku: sku, user: user}
}

func createWholesaleOrderCouponFixture(t *testing.T, db *gorm.DB, productIDs []uint, disabledWholesalePrice bool) coupondomain.Coupon {
	t.Helper()

	scopeIDs, err := json.Marshal(productIDs)
	if err != nil {
		t.Fatalf("marshal coupon scope failed: %v", err)
	}
	coupon := coupondomain.Coupon{
		Code:                   "STACK10",
		Type:                   constants.CouponTypePercent,
		Value:                  money.FromDecimal(decimal.NewFromInt(10)),
		MinAmount:              money.FromDecimal(decimal.Zero),
		MaxDiscount:            money.FromDecimal(decimal.Zero),
		ScopeType:              constants.ScopeTypeProduct,
		ScopeRefIDs:            string(scopeIDs),
		DisabledWholesalePrice: disabledWholesalePrice,
		IsActive:               true,
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
	}
	if err := db.Create(&coupon).Error; err != nil {
		t.Fatalf("create coupon failed: %v", err)
	}
	return coupon
}

func TestBuildOrderResultPrefersWholesaleOverPromotion(t *testing.T) {
	wholesalePrices := productdomain.WholesalePriceTiers{
		{MinQuantity: 5, UnitPrice: money.FromDecimal(decimal.NewFromInt(80))},
	}
	promotionPercent := decimal.NewFromInt(10) // 活动价 90，批发价 80 更便宜。
	fixture := setupWholesaleOrderFixture(t, "wholesale_over_promotion", wholesalePrices, &promotionPercent, nil)

	result, err := fixture.svc.buildOrderResult(orderCreateParams{
		Items: []CreateOrderItem{{ProductID: fixture.product.ID, SKUID: fixture.sku.ID, Quantity: 5}},
	})
	if err != nil {
		t.Fatalf("buildOrderResult failed: %v", err)
	}

	if !result.OriginalAmount.Equal(decimal.NewFromInt(500)) {
		t.Fatalf("expected original amount 500, got %s", result.OriginalAmount.String())
	}
	if !result.WholesaleDiscountAmount.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("expected wholesale discount 100, got %s", result.WholesaleDiscountAmount.String())
	}
	if !result.PromotionDiscountAmount.IsZero() {
		t.Fatalf("expected promotion discount 0, got %s", result.PromotionDiscountAmount.String())
	}
	if !result.TotalAmount.Equal(decimal.NewFromInt(400)) {
		t.Fatalf("expected total 400, got %s", result.TotalAmount.String())
	}
	item := result.Plans[0].Item
	if item.UnitPrice.String() != "80.00" || item.WholesaleDiscount.String() != "100.00" || item.PromotionDiscount.String() != "0.00" {
		t.Fatalf("unexpected item price result: unit=%s wholesale=%s promotion=%s", item.UnitPrice.String(), item.WholesaleDiscount.String(), item.PromotionDiscount.String())
	}
}

func TestBuildOrderResultPrefersPromotionOverWholesale(t *testing.T) {
	wholesalePrices := productdomain.WholesalePriceTiers{
		{MinQuantity: 5, UnitPrice: money.FromDecimal(decimal.NewFromInt(80))},
	}
	promotionPercent := decimal.NewFromInt(30) // 活动价 70，批发价 80 不生效。
	fixture := setupWholesaleOrderFixture(t, "promotion_over_wholesale", wholesalePrices, &promotionPercent, nil)

	result, err := fixture.svc.buildOrderResult(orderCreateParams{
		Items: []CreateOrderItem{{ProductID: fixture.product.ID, SKUID: fixture.sku.ID, Quantity: 5}},
	})
	if err != nil {
		t.Fatalf("buildOrderResult failed: %v", err)
	}

	if !result.PromotionDiscountAmount.Equal(decimal.NewFromInt(150)) {
		t.Fatalf("expected promotion discount 150, got %s", result.PromotionDiscountAmount.String())
	}
	if !result.WholesaleDiscountAmount.IsZero() {
		t.Fatalf("expected wholesale discount 0, got %s", result.WholesaleDiscountAmount.String())
	}
	if !result.TotalAmount.Equal(decimal.NewFromInt(350)) {
		t.Fatalf("expected total 350, got %s", result.TotalAmount.String())
	}
	item := result.Plans[0].Item
	if item.UnitPrice.String() != "70.00" || item.PromotionDiscount.String() != "150.00" || item.WholesaleDiscount.String() != "0.00" {
		t.Fatalf("unexpected item price result: unit=%s wholesale=%s promotion=%s", item.UnitPrice.String(), item.WholesaleDiscount.String(), item.PromotionDiscount.String())
	}
}

func TestBuildOrderResultAppliesCouponAfterBestPromotionOrWholesalePrice(t *testing.T) {
	tests := []struct {
		name              string
		promotionPercent  decimal.Decimal
		wantPromotion     decimal.Decimal
		wantWholesale     decimal.Decimal
		wantCoupon        decimal.Decimal
		wantTotal         decimal.Decimal
		wantUnitPrice     string
		wantCouponPerItem string
	}{
		{
			name:              "wholesale wins before coupon",
			promotionPercent:  decimal.NewFromInt(10),
			wantPromotion:     decimal.Zero,
			wantWholesale:     decimal.NewFromInt(100),
			wantCoupon:        decimal.NewFromInt(40),
			wantTotal:         decimal.NewFromInt(360),
			wantUnitPrice:     "80.00",
			wantCouponPerItem: "40.00",
		},
		{
			name:              "promotion wins before coupon",
			promotionPercent:  decimal.NewFromInt(30),
			wantPromotion:     decimal.NewFromInt(150),
			wantWholesale:     decimal.Zero,
			wantCoupon:        decimal.NewFromInt(35),
			wantTotal:         decimal.NewFromInt(315),
			wantUnitPrice:     "70.00",
			wantCouponPerItem: "35.00",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wholesalePrices := productdomain.WholesalePriceTiers{
				{MinQuantity: 5, UnitPrice: money.FromDecimal(decimal.NewFromInt(80))},
			}
			fixture := setupWholesaleOrderFixture(t, strings.ReplaceAll(tc.name, " ", "_"), wholesalePrices, &tc.promotionPercent, nil)
			coupon := coupondomain.Coupon{
				Code:        "STACK10",
				Type:        constants.CouponTypePercent,
				Value:       money.FromDecimal(decimal.NewFromInt(10)),
				MinAmount:   money.FromDecimal(decimal.Zero),
				MaxDiscount: money.FromDecimal(decimal.Zero),
				ScopeType:   constants.ScopeTypeProduct,
				ScopeRefIDs: fmt.Sprintf("[%d]", fixture.product.ID),
				IsActive:    true,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			if err := fixture.db.Create(&coupon).Error; err != nil {
				t.Fatalf("create coupon failed: %v", err)
			}

			result, err := fixture.svc.buildOrderResult(orderCreateParams{
				CouponCode: "STACK10",
				Items: []CreateOrderItem{
					{ProductID: fixture.product.ID, SKUID: fixture.sku.ID, Quantity: 5},
				},
			})
			if err != nil {
				t.Fatalf("buildOrderResult failed: %v", err)
			}

			if !result.PromotionDiscountAmount.Equal(tc.wantPromotion) {
				t.Fatalf("expected promotion discount %s, got %s", tc.wantPromotion.String(), result.PromotionDiscountAmount.String())
			}
			if !result.WholesaleDiscountAmount.Equal(tc.wantWholesale) {
				t.Fatalf("expected wholesale discount %s, got %s", tc.wantWholesale.String(), result.WholesaleDiscountAmount.String())
			}
			if !result.DiscountAmount.Equal(tc.wantCoupon) {
				t.Fatalf("expected coupon discount %s, got %s", tc.wantCoupon.String(), result.DiscountAmount.String())
			}
			if !result.TotalAmount.Equal(tc.wantTotal) {
				t.Fatalf("expected total %s, got %s", tc.wantTotal.String(), result.TotalAmount.String())
			}
			if len(result.Plans) != 1 {
				t.Fatalf("expected one plan, got %d", len(result.Plans))
			}
			item := result.Plans[0].Item
			if item.UnitPrice.String() != tc.wantUnitPrice || item.CouponDiscount.String() != tc.wantCouponPerItem {
				t.Fatalf("unexpected item result: unit=%s coupon=%s", item.UnitPrice.String(), item.CouponDiscount.String())
			}
		})
	}
}

func TestBuildOrderResultRejectsCouponWhenDisabledForAllWholesaleItems(t *testing.T) {
	wholesalePrices := productdomain.WholesalePriceTiers{
		{MinQuantity: 5, UnitPrice: money.FromDecimal(decimal.NewFromInt(80))},
	}
	promotionPercent := decimal.NewFromInt(10) // 活动价 90，批发价 80 更便宜并实际生效。
	fixture := setupWholesaleOrderFixture(t, "coupon_disabled_all_wholesale", wholesalePrices, &promotionPercent, nil)
	createWholesaleOrderCouponFixture(t, fixture.db, []uint{fixture.product.ID}, true)

	_, err := fixture.svc.buildOrderResult(orderCreateParams{
		CouponCode: "STACK10",
		Items: []CreateOrderItem{
			{ProductID: fixture.product.ID, SKUID: fixture.sku.ID, Quantity: 5},
		},
	})
	if !errors.Is(err, couponcontract.ErrWholesaleDisabled) {
		t.Fatalf("expected couponcontract.ErrWholesaleDisabled, got %v", err)
	}
}

func TestBuildOrderResultExcludesWholesaleItemsWhenCouponDisabledWholesalePrice(t *testing.T) {
	wholesalePrices := productdomain.WholesalePriceTiers{
		{MinQuantity: 5, UnitPrice: money.FromDecimal(decimal.NewFromInt(80))},
	}
	promotionPercent := decimal.NewFromInt(10) // 商品 A 批发价 80 胜出，应被优惠券排除。
	fixture := setupWholesaleOrderFixture(t, "coupon_disabled_mixed_wholesale", wholesalePrices, &promotionPercent, nil)
	now := time.Now()
	productB := productdomain.Product{
		CategoryID:      fixture.product.CategoryID,
		Slug:            "coupon-disabled-mixed-product-b",
		TitleJSON:       jsonmap.JSON{"zh-CN": "非批发商品"},
		PriceAmount:     money.FromDecimal(decimal.NewFromInt(100)),
		PurchaseType:    constants.ProductPurchaseMember,
		FulfillmentType: constants.FulfillmentTypeAuto,
		IsActive:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := fixture.db.Create(&productB).Error; err != nil {
		t.Fatalf("create product b failed: %v", err)
	}
	skuB := productdomain.ProductSKU{
		ProductID:   productB.ID,
		SKUCode:     productdomain.DefaultSKUCode,
		PriceAmount: money.FromDecimal(decimal.NewFromInt(100)),
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := fixture.db.Create(&skuB).Error; err != nil {
		t.Fatalf("create sku b failed: %v", err)
	}
	createWholesaleOrderCouponFixture(t, fixture.db, []uint{fixture.product.ID, productB.ID}, true)

	result, err := fixture.svc.buildOrderResult(orderCreateParams{
		CouponCode: "STACK10",
		Items: []CreateOrderItem{
			{ProductID: fixture.product.ID, SKUID: fixture.sku.ID, Quantity: 5},
			{ProductID: productB.ID, SKUID: skuB.ID, Quantity: 1},
		},
	})
	if err != nil {
		t.Fatalf("buildOrderResult failed: %v", err)
	}
	if !result.WholesaleDiscountAmount.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("expected wholesale discount 100, got %s", result.WholesaleDiscountAmount.String())
	}
	if !result.DiscountAmount.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("expected coupon discount only from product b = 10, got %s", result.DiscountAmount.String())
	}
	if !result.TotalAmount.Equal(decimal.NewFromInt(490)) {
		t.Fatalf("expected total 490, got %s", result.TotalAmount.String())
	}

	byProduct := make(map[uint]orderdomain.OrderItem, len(result.Plans))
	for _, plan := range result.Plans {
		byProduct[plan.Item.ProductID] = plan.Item
	}
	wholesaleItem := byProduct[fixture.product.ID]
	if wholesaleItem.CouponDiscount.String() != "0.00" {
		t.Fatalf("expected wholesale item coupon discount 0, got %s", wholesaleItem.CouponDiscount.String())
	}
	regularItem := byProduct[productB.ID]
	if regularItem.CouponDiscount.String() != "10.00" {
		t.Fatalf("expected regular item coupon discount 10, got %s", regularItem.CouponDiscount.String())
	}
}

func TestBuildOrderResultAppliesMemberDiscountAfterWholesale(t *testing.T) {
	wholesalePrices := productdomain.WholesalePriceTiers{
		{MinQuantity: 5, UnitPrice: money.FromDecimal(decimal.NewFromInt(80))},
	}
	memberRate := decimal.NewFromInt(80) // 批发价 80 后再打 8 折，最终 64。
	fixture := setupWholesaleOrderFixture(t, "member_after_wholesale", wholesalePrices, nil, &memberRate)

	result, err := fixture.svc.buildOrderResult(orderCreateParams{
		UserID: fixture.user.ID,
		Items:  []CreateOrderItem{{ProductID: fixture.product.ID, SKUID: fixture.sku.ID, Quantity: 5}},
	})
	if err != nil {
		t.Fatalf("buildOrderResult failed: %v", err)
	}

	if !result.WholesaleDiscountAmount.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("expected wholesale discount 100, got %s", result.WholesaleDiscountAmount.String())
	}
	if !result.MemberDiscountAmount.Equal(decimal.NewFromInt(80)) {
		t.Fatalf("expected member discount 80, got %s", result.MemberDiscountAmount.String())
	}
	if !result.TotalAmount.Equal(decimal.NewFromInt(320)) {
		t.Fatalf("expected total 320, got %s", result.TotalAmount.String())
	}
	item := result.Plans[0].Item
	if item.UnitPrice.String() != "64.00" || item.WholesaleDiscount.String() != "100.00" || item.MemberDiscount.String() != "80.00" {
		t.Fatalf("unexpected item price result: unit=%s wholesale=%s member=%s", item.UnitPrice.String(), item.WholesaleDiscount.String(), item.MemberDiscount.String())
	}
}

func TestBuildOrderResultMatchesWholesaleByProductQuantityAcrossSKUs(t *testing.T) {
	wholesalePrices := productdomain.WholesalePriceTiers{
		{MinQuantity: 10, UnitPrice: money.FromDecimal(decimal.NewFromInt(80))},
	}
	fixture := setupWholesaleOrderFixture(t, "wholesale_across_skus", wholesalePrices, nil, nil)

	skuB := productdomain.ProductSKU{
		ProductID:   fixture.product.ID,
		SKUCode:     "SKU-B",
		PriceAmount: money.FromDecimal(decimal.NewFromInt(100)),
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := fixture.db.Create(&skuB).Error; err != nil {
		t.Fatalf("create second sku failed: %v", err)
	}

	result, err := fixture.svc.buildOrderResult(orderCreateParams{
		Items: []CreateOrderItem{
			{ProductID: fixture.product.ID, SKUID: fixture.sku.ID, Quantity: 6},
			{ProductID: fixture.product.ID, SKUID: skuB.ID, Quantity: 6},
		},
	})
	if err != nil {
		t.Fatalf("buildOrderResult failed: %v", err)
	}

	if !result.OriginalAmount.Equal(decimal.NewFromInt(1200)) {
		t.Fatalf("expected original amount 1200, got %s", result.OriginalAmount.String())
	}
	if !result.WholesaleDiscountAmount.Equal(decimal.NewFromInt(240)) {
		t.Fatalf("expected wholesale discount 240, got %s", result.WholesaleDiscountAmount.String())
	}
	if !result.TotalAmount.Equal(decimal.NewFromInt(960)) {
		t.Fatalf("expected total 960, got %s", result.TotalAmount.String())
	}
	if len(result.Plans) != 2 {
		t.Fatalf("expected 2 plans, got %d", len(result.Plans))
	}
	for _, plan := range result.Plans {
		if plan.Item.UnitPrice.String() != "80.00" || plan.Item.WholesaleDiscount.String() != "120.00" {
			t.Fatalf("unexpected item price result: sku=%d unit=%s wholesale=%s", plan.Item.SKUID, plan.Item.UnitPrice.String(), plan.Item.WholesaleDiscount.String())
		}
	}
}

// TestBuildOrderResultSkipsWholesaleForCheaperSKU 验证同商品多 SKU 底价不同时的语义：
// 批发门槛按商品总量判定（共享门槛），但批发单价只对「底价高于档位价」的 SKU 生效；
// 底价已低于档位价的 SKU 不会被批发价拉高，各行只计算自己的优惠。
func TestBuildOrderResultSkipsWholesaleForCheaperSKU(t *testing.T) {
	wholesalePrices := productdomain.WholesalePriceTiers{
		{MinQuantity: 10, UnitPrice: money.FromDecimal(decimal.NewFromInt(80))},
	}
	// 默认 SKU 底价 100（高于档位价 80），新增 SKU 底价 50（低于档位价 80）。
	fixture := setupWholesaleOrderFixture(t, "wholesale_skip_cheaper_sku", wholesalePrices, nil, nil)

	cheaperSKU := productdomain.ProductSKU{
		ProductID:   fixture.product.ID,
		SKUCode:     "SKU-CHEAP",
		PriceAmount: money.FromDecimal(decimal.NewFromInt(50)),
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := fixture.db.Create(&cheaperSKU).Error; err != nil {
		t.Fatalf("create cheaper sku failed: %v", err)
	}

	// 6 + 6 = 12 ≥ 10，命中门槛。
	result, err := fixture.svc.buildOrderResult(orderCreateParams{
		Items: []CreateOrderItem{
			{ProductID: fixture.product.ID, SKUID: fixture.sku.ID, Quantity: 6},
			{ProductID: fixture.product.ID, SKUID: cheaperSKU.ID, Quantity: 6},
		},
	})
	if err != nil {
		t.Fatalf("buildOrderResult failed: %v", err)
	}

	// 原价 = 100*6 + 50*6 = 900；批发优惠仅来自底价 100 的 SKU：(100-80)*6 = 120。
	if !result.OriginalAmount.Equal(decimal.NewFromInt(900)) {
		t.Fatalf("expected original amount 900, got %s", result.OriginalAmount.String())
	}
	if !result.WholesaleDiscountAmount.Equal(decimal.NewFromInt(120)) {
		t.Fatalf("expected wholesale discount 120, got %s", result.WholesaleDiscountAmount.String())
	}
	// 实付 = 80*6 + 50*6 = 780。
	if !result.TotalAmount.Equal(decimal.NewFromInt(780)) {
		t.Fatalf("expected total 780, got %s", result.TotalAmount.String())
	}

	bySKU := make(map[uint]orderdomain.OrderItem, len(result.Plans))
	for _, plan := range result.Plans {
		bySKU[plan.Item.SKUID] = plan.Item
	}
	high := bySKU[fixture.sku.ID]
	if high.UnitPrice.String() != "80.00" || high.WholesaleDiscount.String() != "120.00" {
		t.Fatalf("expected high-base SKU to use wholesale 80: unit=%s wholesale=%s", high.UnitPrice.String(), high.WholesaleDiscount.String())
	}
	cheap := bySKU[cheaperSKU.ID]
	if cheap.UnitPrice.String() != "50.00" || cheap.WholesaleDiscount.String() != "0.00" {
		t.Fatalf("expected cheaper SKU to keep base 50 without wholesale: unit=%s wholesale=%s", cheap.UnitPrice.String(), cheap.WholesaleDiscount.String())
	}
}

func TestBuildOrderResultAppliesDifferentWholesalePricesPerSKU(t *testing.T) {
	fixture := setupWholesaleOrderFixture(t, "wholesale_per_sku", nil, nil, nil)
	skuB := productdomain.ProductSKU{
		ProductID:   fixture.product.ID,
		SKUCode:     "SKU-B",
		PriceAmount: money.FromDecimal(decimal.NewFromInt(100)),
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := fixture.db.Create(&skuB).Error; err != nil {
		t.Fatalf("create second sku failed: %v", err)
	}

	wholesalePrices := productdomain.WholesalePriceTiers{
		{SKUID: fixture.sku.ID, SKUCode: fixture.sku.SKUCode, MinQuantity: 5, UnitPrice: money.FromDecimal(decimal.NewFromInt(70))},
		{SKUID: skuB.ID, SKUCode: skuB.SKUCode, MinQuantity: 5, UnitPrice: money.FromDecimal(decimal.NewFromInt(60))},
	}
	if err := fixture.db.Model(&productdomain.Product{}).Where("id = ?", fixture.product.ID).Update("wholesale_prices", wholesalePrices).Error; err != nil {
		t.Fatalf("update wholesale prices failed: %v", err)
	}

	result, err := fixture.svc.buildOrderResult(orderCreateParams{
		Items: []CreateOrderItem{
			{ProductID: fixture.product.ID, SKUID: fixture.sku.ID, Quantity: 5},
			{ProductID: fixture.product.ID, SKUID: skuB.ID, Quantity: 5},
		},
	})
	if err != nil {
		t.Fatalf("buildOrderResult failed: %v", err)
	}

	if !result.OriginalAmount.Equal(decimal.NewFromInt(1000)) {
		t.Fatalf("expected original amount 1000, got %s", result.OriginalAmount.String())
	}
	if !result.WholesaleDiscountAmount.Equal(decimal.NewFromInt(350)) {
		t.Fatalf("expected wholesale discount 350, got %s", result.WholesaleDiscountAmount.String())
	}
	if !result.TotalAmount.Equal(decimal.NewFromInt(650)) {
		t.Fatalf("expected total 650, got %s", result.TotalAmount.String())
	}
	bySKU := make(map[uint]orderdomain.OrderItem, len(result.Plans))
	for _, plan := range result.Plans {
		bySKU[plan.Item.SKUID] = plan.Item
	}
	if item := bySKU[fixture.sku.ID]; item.UnitPrice.String() != "70.00" || item.WholesaleDiscount.String() != "150.00" {
		t.Fatalf("unexpected default sku result: unit=%s wholesale=%s", item.UnitPrice.String(), item.WholesaleDiscount.String())
	}
	if item := bySKU[skuB.ID]; item.UnitPrice.String() != "60.00" || item.WholesaleDiscount.String() != "200.00" {
		t.Fatalf("unexpected skuB result: unit=%s wholesale=%s", item.UnitPrice.String(), item.WholesaleDiscount.String())
	}
}

func TestBuildOrderResultSKUWholesaleDoesNotFallbackToUniversalTier(t *testing.T) {
	fixture := setupWholesaleOrderFixture(t, "wholesale_sku_no_fallback", nil, nil, nil)
	skuB := productdomain.ProductSKU{
		ProductID:   fixture.product.ID,
		SKUCode:     "SKU-B",
		PriceAmount: money.FromDecimal(decimal.NewFromInt(100)),
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := fixture.db.Create(&skuB).Error; err != nil {
		t.Fatalf("create second sku failed: %v", err)
	}

	wholesalePrices := productdomain.WholesalePriceTiers{
		{MinQuantity: 10, UnitPrice: money.FromDecimal(decimal.NewFromInt(80))},
		{SKUID: fixture.sku.ID, SKUCode: fixture.sku.SKUCode, MinQuantity: 10, UnitPrice: money.FromDecimal(decimal.NewFromInt(70))},
	}
	if err := fixture.db.Model(&productdomain.Product{}).Where("id = ?", fixture.product.ID).Update("wholesale_prices", wholesalePrices).Error; err != nil {
		t.Fatalf("update wholesale prices failed: %v", err)
	}

	result, err := fixture.svc.buildOrderResult(orderCreateParams{
		Items: []CreateOrderItem{
			{ProductID: fixture.product.ID, SKUID: fixture.sku.ID, Quantity: 6},
			{ProductID: fixture.product.ID, SKUID: skuB.ID, Quantity: 6},
		},
	})
	if err != nil {
		t.Fatalf("buildOrderResult failed: %v", err)
	}

	if !result.OriginalAmount.Equal(decimal.NewFromInt(1200)) {
		t.Fatalf("expected original amount 1200, got %s", result.OriginalAmount.String())
	}
	if !result.WholesaleDiscountAmount.Equal(decimal.NewFromInt(120)) {
		t.Fatalf("expected wholesale discount 120, got %s", result.WholesaleDiscountAmount.String())
	}
	if !result.TotalAmount.Equal(decimal.NewFromInt(1080)) {
		t.Fatalf("expected total 1080, got %s", result.TotalAmount.String())
	}
	bySKU := make(map[uint]orderdomain.OrderItem, len(result.Plans))
	for _, plan := range result.Plans {
		bySKU[plan.Item.SKUID] = plan.Item
	}
	if item := bySKU[fixture.sku.ID]; item.UnitPrice.String() != "100.00" || item.WholesaleDiscount.String() != "0.00" {
		t.Fatalf("expected specific SKU to skip universal fallback: unit=%s wholesale=%s", item.UnitPrice.String(), item.WholesaleDiscount.String())
	}
	if item := bySKU[skuB.ID]; item.UnitPrice.String() != "80.00" || item.WholesaleDiscount.String() != "120.00" {
		t.Fatalf("expected skuB to use universal tier: unit=%s wholesale=%s", item.UnitPrice.String(), item.WholesaleDiscount.String())
	}
}

type syncingStockEnsurer func()

func (f syncingStockEnsurer) EnsureUpstreamStockForOrder(uint, int) error {
	f()
	return nil
}

// 下单前的上游库存检查可能实时同步并恢复交付表单，校验必须使用同步后的 schema。
func TestBuildOrderResultValidatesManualFormRestoredByUpstreamSync(t *testing.T) {
	fixture := setupWholesaleOrderFixture(t, "upstream_form_resync", nil, nil, nil)
	fixture.product.FulfillmentType = constants.FulfillmentTypeUpstream
	fixture.product.ManualFormSchemaJSON = jsonmap.JSON{}
	if err := fixture.db.Save(&fixture.product).Error; err != nil {
		t.Fatalf("save product failed: %v", err)
	}
	schema := jsonmap.JSON{"fields": []interface{}{map[string]interface{}{
		"key": "x_handle", "type": "text", "required": true,
	}}}
	fixture.svc.SetProductMappingService(syncingStockEnsurer(func() {
		if err := fixture.db.Model(&productdomain.Product{ID: fixture.product.ID}).Update("ManualFormSchemaJSON", schema).Error; err != nil {
			t.Fatalf("simulate sync failed: %v", err)
		}
	}))

	if _, err := fixture.svc.buildOrderResult(orderCreateParams{
		Items: []CreateOrderItem{{ProductID: fixture.product.ID, SKUID: fixture.sku.ID, Quantity: 1}},
	}); err == nil {
		t.Fatal("expected empty manual form to be rejected after sync restored a required schema")
	}

	result, err := fixture.svc.buildOrderResult(orderCreateParams{
		Items:          []CreateOrderItem{{ProductID: fixture.product.ID, SKUID: fixture.sku.ID, Quantity: 1}},
		ManualFormData: map[string]jsonmap.JSON{fmt.Sprintf("%d", fixture.product.ID): {"x_handle": "fixture_user"}},
	})
	if err != nil {
		t.Fatalf("buildOrderResult failed: %v", err)
	}
	if got := result.Plans[0].Item.ManualFormSubmissionJSON["x_handle"]; got != "fixture_user" {
		t.Fatalf("expected submission snapshot, got %#v", result.Plans[0].Item.ManualFormSubmissionJSON)
	}
}

func TestAPIPlanCategoryRejectsGuestOrderWithLegacyGuestProductFlag(t *testing.T) {
	fixture := setupWholesaleOrderFixture(t, "api_plan_member_gate", nil, nil, nil)
	root := categorydomain.Category{Slug: "plans", NameJSON: jsonmap.JSON{"zh-CN": "API"}, IsActive: true}
	if err := fixture.db.Create(&root).Error; err != nil {
		t.Fatal(err)
	}
	child := categorydomain.Category{Slug: "plans-personal", ParentID: root.ID, NameJSON: jsonmap.JSON{"zh-CN": "个人"}, IsActive: true}
	if err := fixture.db.Create(&child).Error; err != nil {
		t.Fatal(err)
	}
	deep := categorydomain.Category{Slug: "api-dedicated", ParentID: child.ID, NameJSON: jsonmap.JSON{"zh-CN": "专用"}, IsActive: true}
	if err := fixture.db.Create(&deep).Error; err != nil {
		t.Fatal(err)
	}
	originalCategory := fixture.product.CategoryID
	for _, categoryID := range []uint{root.ID, child.ID, deep.ID} {
		if err := fixture.db.Model(&fixture.product).Updates(map[string]interface{}{"category_id": categoryID, "purchase_type": constants.ProductPurchaseGuest}).Error; err != nil {
			t.Fatal(err)
		}
		input := orderCreateParams{IsGuest: true, GuestEmail: "buyer@example.test", GuestPassword: "test-password", Items: []CreateOrderItem{{ProductID: fixture.product.ID, SKUID: fixture.sku.ID, Quantity: 1}}}
		if _, err := fixture.svc.buildOrderResult(input); !errors.Is(err, ErrProductPurchaseNotAllowed) {
			t.Fatalf("API category %d accepted guest request or returned wrong error: %v", categoryID, err)
		}
		input.IsGuest = false
		input.UserID = 1
		if _, err := fixture.svc.buildOrderResult(input); err == nil || errors.Is(err, ErrProductPurchaseNotAllowed) {
			t.Fatalf("API member must pass identity gate but cannot buy a plan without a verified entitlement: %v", err)
		}
	}
	if err := fixture.db.Model(&fixture.product).Update("category_id", originalCategory).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.svc.buildOrderResult(orderCreateParams{IsGuest: true, GuestEmail: "buyer@example.test", GuestPassword: "test-password", Items: []CreateOrderItem{{ProductID: fixture.product.ID, SKUID: fixture.sku.ID, Quantity: 1}}}); err != nil {
		t.Fatalf("ordinary guest product was blocked: %v", err)
	}
}

type orderPlanPreflight struct {
	calls      int
	userID     uint
	localSKUID uint
	raw        string
	err        error
}

func (*orderPlanPreflight) EnsureUpstreamStockForOrder(uint, int) error { return nil }
func (*orderPlanPreflight) Enabled() bool                               { return true }
func (*orderPlanPreflight) EnqueueTimeoutCancel(uint, time.Duration) error {
	panic("preflight failure reached queue")
}
func (*orderPlanPreflight) EnqueueStatusEmail(uint, string) error {
	panic("preflight failure reached queue")
}
func (p *orderPlanPreflight) PreparePlanCheckout(localSKUID, userID uint, quantity int, raw json.RawMessage, hash string) (mappingcontract.PlanCheckoutRoute, error) {
	p.calls++
	p.userID, p.localSKUID, p.raw = userID, localSKUID, string(raw)
	return mappingcontract.PlanCheckoutRoute{ConnectionID: 77, UpstreamSKUID: 901}, p.err
}

func TestAPIPlanOrderPreflightFreezesExactTermsAndMemberBeforePersistence(t *testing.T) {
	fixture := setupWholesaleOrderFixture(t, "api_plan_preflight", nil, nil, nil)
	fixture.user = userdomain.User{Email: "api-plan-native-member@example.test", PasswordHash: "hash", Status: constants.UserStatusActive}
	if err := fixture.db.Create(&fixture.user).Error; err != nil {
		t.Fatal(err)
	}
	var contract jsonmap.ExactJSON
	if err := json.Unmarshal([]byte(`{"kind":"plan","issue_api_key":true,"plan":{"quota":9007199254740993},"future_field":"preserved"}`), &contract); err != nil {
		t.Fatal(err)
	}
	if err := fixture.db.Model(&fixture.sku).Updates(map[string]interface{}{"entitlement_contract_json": contract, "entitlement_contract_hash": strings.Repeat("a", 64)}).Error; err != nil {
		t.Fatal(err)
	}
	// The entitlement remains a plan even if operations moves its category or
	// leaves the historical purchase flag set to guest.
	if err := fixture.db.Model(&fixture.product).Update("purchase_type", constants.ProductPurchaseGuest).Error; err != nil {
		t.Fatal(err)
	}
	preflight := &orderPlanPreflight{}
	fixture.svc.SetProductMappingService(preflight)
	fixture.svc.queueClient = preflight
	input := orderCreateParams{UserID: fixture.user.ID, Items: []CreateOrderItem{{ProductID: fixture.product.ID, SKUID: fixture.sku.ID, Quantity: 1}}}
	result, err := fixture.svc.buildOrderResult(input)
	if err != nil {
		t.Fatal(err)
	}
	if preflight.calls != 1 || preflight.userID != fixture.user.ID || preflight.localSKUID != fixture.sku.ID || !strings.Contains(preflight.raw, "9007199254740993") || !strings.Contains(preflight.raw, "future_field") {
		t.Fatalf("wrong native preflight: %#v", preflight)
	}
	snapshot := result.Plans[0].Item.SKUSnapshotJSON
	if snapshot["api_plan"] != true || snapshot["plan_checkout_connection_id"] != "77" || snapshot["plan_checkout_upstream_sku_id"] != "901" || snapshot["plan_checkout_entitlement_raw"] != preflight.raw {
		t.Fatalf("frozen plan route/terms missing: %#v", snapshot)
	}
	// Generic order SQL serialization must retain the exact signed bytes string.
	value, _ := snapshot.Value()
	var stored jsonmap.JSON
	if err := stored.Scan(value); err != nil {
		t.Fatal(err)
	}
	if stored["plan_checkout_entitlement_raw"] != preflight.raw {
		t.Fatal("order snapshot changed exact signed contract")
	}
	preflight.err = mappingcontract.ErrPlanAccountLinkRequired
	// This fixture has no order store. Rejection must happen before any access
	// to order persistence/transaction machinery in the real creation path.
	if _, err := fixture.svc.createOrder(input); !errors.Is(err, mappingcontract.ErrPlanAccountLinkRequired) {
		t.Fatalf("unlinked member accepted: %v", err)
	}
	input.IsGuest, input.GuestEmail, input.GuestPassword, input.UserID = true, "buyer@example.test", "test-password", 0
	before := preflight.calls
	if _, err := fixture.svc.buildOrderResult(input); !errors.Is(err, ErrProductPurchaseNotAllowed) || preflight.calls != before {
		t.Fatalf("guest bypassed moved-category plan: %v", err)
	}
}
