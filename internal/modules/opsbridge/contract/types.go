package contract

import (
	"errors"
	"time"
)

var (
	ErrNotConfigured = errors.New("opsbridge: capability not configured")
	ErrNotFound      = errors.New("opsbridge: record not found")
	ErrInvalidInput  = errors.New("opsbridge: invalid input")
	ErrConflict      = errors.New("opsbridge: conflict")
)

type Payment struct {
	ID           uint       `json:"id"`
	ProviderType string     `json:"provider_type,omitempty"`
	ChannelType  string     `json:"channel_type,omitempty"`
	Status       string     `json:"status"`
	Amount       string     `json:"amount"`
	Currency     string     `json:"currency"`
	ProviderRef  string     `json:"provider_ref,omitempty"`
	PaidAt       *time.Time `json:"paid_at,omitempty"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
type Refund struct {
	ID               uint      `json:"id"`
	OperationID      *string   `json:"operation_id,omitempty"`
	Type             string    `json:"type"`
	Amount           string    `json:"amount"`
	Currency         string    `json:"currency"`
	SettlementStatus string    `json:"settlement_status"`
	UpdatedAt        time.Time `json:"updated_at"`
}
type Procurement struct {
	LocalOrderID    uint      `json:"local_order_id"`
	ID              uint      `json:"id"`
	Status          string    `json:"status"`
	UpstreamOrderNo string    `json:"upstream_order_no,omitempty"`
	UpdatedAt       time.Time `json:"updated_at"`
}
type Reservation struct {
	OrderItemID     uint      `json:"order_item_id"`
	LocalOrderID    uint      `json:"local_order_id"`
	HoldID          string    `json:"hold_id"`
	OrderRef        string    `json:"order_ref"`
	Version         string    `json:"version"`
	Status          string    `json:"status"`
	BookStatus      string    `json:"book_status"`
	ReadyForPayment bool      `json:"ready_for_payment"`
	ExpiresAt       time.Time `json:"expires_at"`
	AttentionCode   string    `json:"attention_code,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
type Order struct {
	Reservations       []Reservation `json:"reservations"`
	Children           []Order       `json:"children"`
	Procurements       []Procurement `json:"procurements"`
	ID                 uint          `json:"id"`
	OrderNo            string        `json:"order_no"`
	ParentID           *uint         `json:"parent_id,omitempty"`
	MerchantCustomerID string        `json:"merchant_customer_id,omitempty"`
	Status             string        `json:"status"`
	Currency           string        `json:"currency"`
	OriginalAmount     string        `json:"original_amount,omitempty"`
	DiscountAmount     string        `json:"discount_amount,omitempty"`
	CouponID           *uint         `json:"coupon_id,omitempty"`
	TotalAmount        string        `json:"total_amount"`
	RefundedAmount     string        `json:"refunded_amount"`
	PaymentStatus      string        `json:"payment_status"`
	FulfillmentStatus  string        `json:"fulfillment_status"`
	EntitlementStatus  string        `json:"entitlement_status"`
	Payments           []Payment     `json:"payments"`
	Refunds            []Refund      `json:"refunds"`
	Procurement        *Procurement  `json:"procurement,omitempty"`
	PaidAt             *time.Time    `json:"paid_at,omitempty"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
}
type OrdersResult struct {
	Source      string    `json:"source"`
	CollectedAt time.Time `json:"collected_at"`
	Freshness   string    `json:"freshness"`
	Orders      []Order   `json:"orders"`
	Total       int64     `json:"total"`
	Page        int       `json:"page"`
	PageSize    int       `json:"page_size"`
}
type Overview struct {
	Source         string           `json:"source"`
	CollectedAt    time.Time        `json:"collected_at"`
	Freshness      string           `json:"freshness"`
	TotalOrders    int64            `json:"total_orders"`
	OrdersByStatus map[string]int64 `json:"orders_by_status"`
	Capabilities   []string         `json:"capabilities"`
}

// Storefront catalog & coupon management DTOs for OPS high-frequency linkage.

type Category struct {
	ID        uint              `json:"id"`
	ParentID  uint              `json:"parent_id"`
	Slug      string            `json:"slug"`
	Name      map[string]string `json:"name"`
	SortOrder int               `json:"sort_order"`
	IsActive  bool              `json:"is_active"`
	CreatedAt time.Time         `json:"created_at"`
}

type CategoriesResult struct {
	Source      string     `json:"source"`
	CollectedAt time.Time  `json:"collected_at"`
	Freshness   string     `json:"freshness"`
	Categories  []Category `json:"categories"`
}

type UpdateCategoryInput struct {
	SortOrder *int  `json:"sort_order,omitempty"`
	IsActive  *bool `json:"is_active,omitempty"`
}

type ProductSKU struct {
	ID                uint      `json:"id"`
	ProductID         uint      `json:"product_id"`
	SKUCode           string    `json:"sku_code"`
	PriceAmount       string    `json:"price_amount"`
	CostPriceAmount   string    `json:"cost_price_amount"`
	ManualStockTotal  int       `json:"manual_stock_total"`
	ManualStockLocked int       `json:"manual_stock_locked"`
	ManualStockSold   int       `json:"manual_stock_sold"`
	IsActive          bool      `json:"is_active"`
	SortOrder         int       `json:"sort_order"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type Product struct {
	ID                uint              `json:"id"`
	CategoryID        uint              `json:"category_id"`
	Slug              string            `json:"slug"`
	Title             map[string]string `json:"title"`
	Description       map[string]string `json:"description,omitempty"`
	PriceAmount       string            `json:"price_amount"`
	CostPriceAmount   string            `json:"cost_price_amount"`
	PurchaseType      string            `json:"purchase_type"`
	FulfillmentType   string            `json:"fulfillment_type"`
	ManualStockTotal  int               `json:"manual_stock_total"`
	ManualStockLocked int               `json:"manual_stock_locked"`
	ManualStockSold   int               `json:"manual_stock_sold"`
	IsMapped          bool              `json:"is_mapped"`
	IsActive          bool              `json:"is_active"`
	SortOrder         int               `json:"sort_order"`
	SKUs              []ProductSKU      `json:"skus"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

type ProductsResult struct {
	Source      string    `json:"source"`
	CollectedAt time.Time `json:"collected_at"`
	Freshness   string    `json:"freshness"`
	Products    []Product `json:"products"`
	Total       int64     `json:"total"`
	Page        int       `json:"page"`
	PageSize    int       `json:"page_size"`
}

type CreateProductInput struct {
	CategoryID       uint              `json:"category_id"`
	Slug             string            `json:"slug"`
	Title            map[string]string `json:"title"`
	Description      map[string]string `json:"description,omitempty"`
	PriceAmount      string            `json:"price_amount"`
	CostPriceAmount  string            `json:"cost_price_amount,omitempty"`
	PurchaseType     string            `json:"purchase_type,omitempty"`
	FulfillmentType  string            `json:"fulfillment_type,omitempty"`
	ManualStockTotal *int              `json:"manual_stock_total,omitempty"`
	SKUCode          string            `json:"sku_code,omitempty"`
	SortOrder        int               `json:"sort_order,omitempty"`
	IsActive         *bool             `json:"is_active,omitempty"`
}

type UpdateProductInput struct {
	CategoryID       *uint             `json:"category_id,omitempty"`
	Slug             *string           `json:"slug,omitempty"`
	Title            map[string]string `json:"title,omitempty"`
	Description      map[string]string `json:"description,omitempty"`
	PriceAmount      *string           `json:"price_amount,omitempty"`
	CostPriceAmount  *string           `json:"cost_price_amount,omitempty"`
	PurchaseType     *string           `json:"purchase_type,omitempty"`
	FulfillmentType  *string           `json:"fulfillment_type,omitempty"`
	ManualStockTotal *int              `json:"manual_stock_total,omitempty"`
	IsActive         *bool             `json:"is_active,omitempty"`
	SortOrder        *int              `json:"sort_order,omitempty"`
}

type CreateSKUInput struct {
	SKUCode          string `json:"sku_code"`
	PriceAmount      string `json:"price_amount"`
	CostPriceAmount  string `json:"cost_price_amount,omitempty"`
	ManualStockTotal *int   `json:"manual_stock_total,omitempty"`
	SortOrder        int    `json:"sort_order,omitempty"`
	IsActive         *bool  `json:"is_active,omitempty"`
}

type UpdateSKUInput struct {
	SKUCode          *string `json:"sku_code,omitempty"`
	PriceAmount      *string `json:"price_amount,omitempty"`
	CostPriceAmount  *string `json:"cost_price_amount,omitempty"`
	ManualStockTotal *int    `json:"manual_stock_total,omitempty"`
	SortOrder        *int    `json:"sort_order,omitempty"`
	IsActive         *bool   `json:"is_active,omitempty"`
}

type ConnectionPricing struct {
	ID                 uint      `json:"id"`
	Name               string    `json:"name"`
	Protocol           string    `json:"protocol"`
	Status             string    `json:"status"`
	ExchangeRate       string    `json:"exchange_rate"`
	PriceMarkupPercent string    `json:"price_markup_percent"`
	PriceRoundingMode  string    `json:"price_rounding_mode"`
	AutoSyncPrice      bool      `json:"auto_sync_price"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type PricingSettings struct {
	Source                 string              `json:"source"`
	CollectedAt            time.Time           `json:"collected_at"`
	Freshness              string              `json:"freshness"`
	SiteCurrency           string              `json:"site_currency"`
	DefaultDisplayCurrency string              `json:"default_display_currency"`
	USDCNYRate             string              `json:"usd_cny_rate"`
	RateUpdatedAt          string              `json:"rate_updated_at,omitempty"`
	Connections            []ConnectionPricing `json:"connections"`
}

type UpdatePricingInput struct {
	SiteCurrency           *string  `json:"site_currency,omitempty"`
	DefaultDisplayCurrency *string  `json:"default_display_currency,omitempty"`
	USDCNYRate             *string  `json:"usd_cny_rate,omitempty"`
	ConnectionID           *uint    `json:"connection_id,omitempty"`
	ExchangeRate           *float64 `json:"exchange_rate,omitempty"`
	PriceMarkupPercent     *float64 `json:"price_markup_percent,omitempty"`
	PriceRoundingMode      *string  `json:"price_rounding_mode,omitempty"`
	AutoSyncPrice          *bool    `json:"auto_sync_price,omitempty"`
}

// PricingStore abstracts storefront currency/display-rate settings and upstream
// connection exchange-rate/markup settings without exposing secrets.
type PricingStore interface {
	GetSitePricing() (siteCurrency, defaultDisplayCurrency, usdCnyRate, rateUpdatedAt string, err error)
	UpdateSitePricing(siteCurrency, defaultDisplayCurrency, usdCnyRate *string) error
	ListConnectionPricing() ([]ConnectionPricing, error)
	UpdateConnectionPricing(id uint, exchangeRate, markupPercent *float64, roundingMode *string, autoSyncPrice *bool) error
}

type Coupon struct {
	ID           uint       `json:"id"`
	Code         string     `json:"code"`
	Type         string     `json:"type"`
	Value        string     `json:"value"`
	MinAmount    string     `json:"min_amount"`
	MaxDiscount  string     `json:"max_discount"`
	UsageLimit   int        `json:"usage_limit"`
	UsedCount    int        `json:"used_count"`
	PerUserLimit int        `json:"per_user_limit"`
	ScopeType    string     `json:"scope_type"`
	ScopeRefIDs  string     `json:"scope_ref_ids"`
	StartsAt     *time.Time `json:"starts_at,omitempty"`
	EndsAt       *time.Time `json:"ends_at,omitempty"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type CouponsResult struct {
	Source      string    `json:"source"`
	CollectedAt time.Time `json:"collected_at"`
	Freshness   string    `json:"freshness"`
	Coupons     []Coupon  `json:"coupons"`
	Total       int64     `json:"total"`
	Page        int       `json:"page"`
	PageSize    int       `json:"page_size"`
}

type CreateCouponInput struct {
	Code         string     `json:"code"`
	Type         string     `json:"type"`
	Value        string     `json:"value"`
	MinAmount    string     `json:"min_amount,omitempty"`
	MaxDiscount  string     `json:"max_discount,omitempty"`
	UsageLimit   int        `json:"usage_limit,omitempty"`
	PerUserLimit int        `json:"per_user_limit,omitempty"`
	ScopeRefIDs  []uint     `json:"scope_ref_ids,omitempty"`
	StartsAt     *time.Time `json:"starts_at,omitempty"`
	EndsAt       *time.Time `json:"ends_at,omitempty"`
	IsActive     *bool      `json:"is_active,omitempty"`
}

type UpdateCouponStatusInput struct {
	IsActive bool `json:"is_active"`
}

type Reader interface {
	Orders(page, pageSize int) (*OrdersResult, error)
	Overview() (*Overview, error)
}

type Bridge interface {
	Reader
	Categories() (*CategoriesResult, error)
	UpdateCategory(id string, input UpdateCategoryInput) (*Category, error)
	Products(page, pageSize int, categoryID, search string) (*ProductsResult, error)
	CreateProduct(input CreateProductInput) (*Product, error)
	UpdateProduct(id string, input UpdateProductInput) (*Product, error)
	CreateSKU(productID uint, input CreateSKUInput) (*ProductSKU, error)
	UpdateSKU(id uint, input UpdateSKUInput) (*ProductSKU, error)
	Coupons(page, pageSize int, code string, isActive *bool) (*CouponsResult, error)
	CreateCoupon(input CreateCouponInput) (*Coupon, error)
	UpdateCouponStatus(id uint, active bool) (*Coupon, error)
	Pricing() (*PricingSettings, error)
	UpdatePricing(input UpdatePricingInput) (*PricingSettings, error)
}
