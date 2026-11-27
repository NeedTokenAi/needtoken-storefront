package contract

import (
	"errors"
	"time"

	productcontract "github.com/dujiao-next/internal/modules/catalog/product/contract"
)

var (
	ErrNotConfigured = errors.New("opsbridge: capability not configured")
	ErrNotFound      = errors.New("opsbridge: record not found")
	ErrInvalidInput  = errors.New("opsbridge: invalid input")
	ErrConflict      = errors.New("opsbridge: conflict")
)

// ConflictError wraps ErrConflict with a stable machine-readable error_code
// for HTTP 409 responses on catalog mutation endpoints.
type ConflictError struct {
	Code string
}

func (e *ConflictError) Error() string {
	return "opsbridge: conflict: " + e.Code
}

func (e *ConflictError) Unwrap() error {
	return ErrConflict
}

var (
	ErrCategoryInUse            = &ConflictError{Code: "category_in_use"}
	ErrProductHasStock          = &ConflictError{Code: "product_has_stock"}
	ErrProductHasOrderRecord    = &ConflictError{Code: "product_has_order_record"}
	ErrProductRequiresActiveSKU = &ConflictError{Code: "product_requires_active_sku"}
	ErrSKUHasCardSecretStock    = &ConflictError{Code: "sku_has_card_secret_stock"}
)

type Payment struct {
	ID                 uint           `json:"id"`
	ProviderType       string         `json:"provider_type,omitempty"`
	ChannelType        string         `json:"channel_type,omitempty"`
	PaymentEnvironment string         `json:"payment_environment,omitempty"`
	Status             string         `json:"status"`
	Amount             string         `json:"amount"`
	FeeAmount          string         `json:"fee_amount,omitempty"`
	Currency           string         `json:"currency"`
	ExceptionCode      string         `json:"exception_code,omitempty"`
	GatewayOrderNo     string         `json:"gateway_order_no,omitempty"`
	ProviderRef        string         `json:"provider_ref,omitempty"`
	CallbackAt         *time.Time     `json:"callback_at,omitempty"`
	CallbackSummary    map[string]any `json:"callback_summary,omitempty"`
	PaidAt             *time.Time     `json:"paid_at,omitempty"`
	UpdatedAt          time.Time      `json:"updated_at"`
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
	ErrorMessage    string    `json:"error_message,omitempty"`
	RetryCount      int       `json:"retry_count,omitempty"`
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
type OrderItemSummary struct {
	ID              uint              `json:"id"`
	OrderID         uint              `json:"order_id,omitempty"`
	ProductID       uint              `json:"product_id"`
	CategoryID      uint              `json:"category_id,omitempty"`
	SKUID           uint              `json:"sku_id,omitempty"`
	SKUCode         string            `json:"sku_code,omitempty"`
	Title           map[string]string `json:"title,omitempty"`
	FulfillmentType string            `json:"fulfillment_type,omitempty"`
	Quantity        int               `json:"quantity"`
	UnitPrice       string            `json:"unit_price,omitempty"`
	TotalPrice      string            `json:"total_price,omitempty"`
	Tags            []string          `json:"tags,omitempty"`
}
type Order struct {
	Reservations       []Reservation      `json:"reservations"`
	Children           []Order            `json:"children"`
	Procurements       []Procurement      `json:"procurements"`
	Items              []OrderItemSummary `json:"items"`
	ID                 uint               `json:"id"`
	OrderNo            string             `json:"order_no"`
	ParentID           *uint              `json:"parent_id,omitempty"`
	UserID             uint               `json:"user_id,omitempty"`
	BuyerType          string             `json:"buyer_type,omitempty"`
	BuyerEmail         string             `json:"buyer_email,omitempty"`
	BuyerName          string             `json:"buyer_name,omitempty"`
	MerchantCustomerID string             `json:"merchant_customer_id,omitempty"`
	SourceDomain       string             `json:"source_domain,omitempty"`
	ClientIP           string             `json:"client_ip,omitempty"`
	UserAgent          string             `json:"user_agent,omitempty"`
	ClientFingerprint  string             `json:"client_fingerprint,omitempty"`
	Status             string             `json:"status"`
	Currency           string             `json:"currency"`
	OriginalAmount     string             `json:"original_amount,omitempty"`
	DiscountAmount     string             `json:"discount_amount,omitempty"`
	CouponID           *uint              `json:"coupon_id,omitempty"`
	TotalAmount        string             `json:"total_amount"`
	WalletPaidAmount   string             `json:"wallet_paid_amount,omitempty"`
	OnlinePaidAmount   string             `json:"online_paid_amount,omitempty"`
	RefundedAmount     string             `json:"refunded_amount"`
	PaymentStatus      string             `json:"payment_status"`
	FulfillmentStatus  string             `json:"fulfillment_status"`
	EntitlementStatus  string             `json:"entitlement_status"`
	HasException       bool               `json:"has_exception"`
	ExceptionReasons   []string           `json:"exception_reasons,omitempty"`
	Payments           []Payment          `json:"payments"`
	Refunds            []Refund           `json:"refunds"`
	Procurement        *Procurement       `json:"procurement,omitempty"`
	PaidAt             *time.Time         `json:"paid_at,omitempty"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
}
type OrdersFilter struct {
	Page            int
	PageSize        int
	Status          string
	OrderNo         string
	Keyword         string
	CategoryID      uint
	FulfillmentType string
	BuyerType       string
	PaymentChannel  string
	ClientIP        string
	ExceptionOnly   bool
	CreatedFrom     *time.Time
	CreatedTo       *time.Time
}
type PendingUnifiedOp struct {
	OpID      string    `json:"op_id"`
	UserID    uint      `json:"user_id"`
	Kind      string    `json:"kind"`
	Status    string    `json:"status"`
	Amount    string    `json:"amount"`
	Currency  string    `json:"currency"`
	Attempts  int       `json:"attempts"`
	LastError string    `json:"last_error,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}
type UnifiedOpReader interface {
	ListPendingUnifiedOpsForOps(limit int) ([]PendingUnifiedOp, error)
}
type OrdersResult struct {
	Source            string             `json:"source"`
	CollectedAt       time.Time          `json:"collected_at"`
	Freshness         string             `json:"freshness"`
	Orders            []Order            `json:"orders"`
	Total             int64              `json:"total"`
	Page              int                `json:"page"`
	PageSize          int                `json:"page_size"`
	PendingUnifiedOps []PendingUnifiedOp `json:"pending_unified_ops,omitempty"`
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
	Icon      string            `json:"icon,omitempty"`
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

type CreateCategoryInput struct {
	ParentID  uint              `json:"parent_id"`
	Slug      string            `json:"slug"`
	Name      map[string]string `json:"name"`
	Icon      string            `json:"icon,omitempty"`
	SortOrder int               `json:"sort_order,omitempty"`
	IsActive  *bool             `json:"is_active,omitempty"`
}

type UpdateCategoryInput struct {
	ParentID  *uint             `json:"parent_id,omitempty"`
	Slug      *string           `json:"slug,omitempty"`
	Name      map[string]string `json:"name,omitempty"`
	Icon      *string           `json:"icon,omitempty"`
	SortOrder *int              `json:"sort_order,omitempty"`
	IsActive  *bool             `json:"is_active,omitempty"`
}

type DeleteResult struct {
	Deleted bool `json:"deleted"`
	ID      uint `json:"id"`
}

type BatchDeleteProductsInput struct {
	IDs []uint `json:"ids"`
}

type BatchDeleteError struct {
	ID    uint   `json:"id"`
	Error string `json:"error"`
}

type BatchDeleteResult struct {
	Total     int                `json:"total"`
	Succeeded int                `json:"succeeded"`
	Failed    int                `json:"failed"`
	Errors    []BatchDeleteError `json:"errors"`
}

// CategoryStore delegates category CRUD and two-level tree guards to the native CategoryService.
type CategoryStore interface {
	Create(parentID uint, slug string, name map[string]any, icon string, sortOrder int, isActive bool) (*Category, error)
	Update(id string, parentID uint, slug string, name map[string]any, icon string, sortOrder int, isActive *bool) (*Category, error)
	Delete(id string) error
}

// ProductDeleteStore delegates safe product deletion to the native ProductAdminService.
type ProductDeleteStore interface {
	Delete(id string) error
}

// CardSecretStockCounter checks per-SKU card secret usage before deleting a SKU.
type CardSecretStockCounter interface {
	CountByProduct(productID, skuID uint) (total, available, used int64, err error)
}

// SKUDeleteUnitOfWork runs single-SKU hard delete and product wholesale/price sync
// inside one transaction handle so SQLite MaxOpenConns=1 never self-deadlocks.
type SKUDeleteUnitOfWork interface {
	WithinTransaction(fn func(products productcontract.Repository, skus productcontract.SKURepository) error) error
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

type GiftCard struct {
	ID             uint       `json:"id"`
	BatchID        *uint      `json:"batch_id,omitempty"`
	BatchNo        string     `json:"batch_no,omitempty"`
	Name           string     `json:"name"`
	Code           string     `json:"code"`
	Amount         string     `json:"amount"`
	Currency       string     `json:"currency"`
	Status         string     `json:"status"`
	IsExpired      bool       `json:"is_expired"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	RedeemedAt     *time.Time `json:"redeemed_at,omitempty"`
	RedeemedUserID *uint      `json:"redeemed_user_id,omitempty"`
	WalletTxnID    *uint      `json:"wallet_txn_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type GiftCardBatch struct {
	ID        uint       `json:"id"`
	BatchNo   string     `json:"batch_no"`
	Name      string     `json:"name"`
	Amount    string     `json:"amount"`
	Currency  string     `json:"currency"`
	Quantity  int        `json:"quantity"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type GiftCardsResult struct {
	Source      string     `json:"source"`
	CollectedAt time.Time  `json:"collected_at"`
	Freshness   string     `json:"freshness"`
	GiftCards   []GiftCard `json:"gift_cards"`
	Cards       []GiftCard `json:"cards"`
	Total       int64      `json:"total"`
	Page        int        `json:"page"`
	PageSize    int        `json:"page_size"`
}

type GenerateGiftCardsInput struct {
	Name      string `json:"name"`
	Quantity  int    `json:"quantity"`
	Amount    string `json:"amount"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

type GenerateGiftCardsResult struct {
	Source      string        `json:"source"`
	CollectedAt time.Time     `json:"collected_at"`
	Freshness   string        `json:"freshness"`
	Batch       GiftCardBatch `json:"batch"`
	BatchNo     string        `json:"batch_no"`
	Created     int           `json:"created"`
	Cards       []GiftCard    `json:"cards"`
}

type AcquiringChannel struct {
	ID                 uint              `json:"id"`
	OpsChannelRef      string            `json:"ops_channel_ref,omitempty"`
	Name               string            `json:"name"`
	Icon               string            `json:"icon,omitempty"`
	ProviderType       string            `json:"provider_type"`
	ChannelType        string            `json:"channel_type"`
	InteractionMode    string            `json:"interaction_mode"`
	PaymentEnvironment string            `json:"payment_environment"`
	FeeRate            string            `json:"fee_rate"`
	FixedFee           string            `json:"fixed_fee"`
	MinAmount          string            `json:"min_amount"`
	MaxAmount          string            `json:"max_amount"`
	PaymentRoles       []string          `json:"payment_roles,omitempty"`
	PaymentTypes       []string          `json:"payment_types,omitempty"`
	ConfigSummary      map[string]string `json:"config_summary,omitempty"`
	IsActive           bool              `json:"is_active"`
	SortOrder          int               `json:"sort_order"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

type AcquiringChannelsResult struct {
	Source      string             `json:"source"`
	CollectedAt time.Time          `json:"collected_at"`
	Freshness   string             `json:"freshness"`
	Channels    []AcquiringChannel `json:"channels"`
	Total       int64              `json:"total"`
}

type UpsertAcquiringChannelInput struct {
	ID                 uint           `json:"id,omitempty"`
	OpsChannelRef      string         `json:"ops_channel_ref"`
	Name               string         `json:"name"`
	Icon               string         `json:"icon,omitempty"`
	ProviderType       string         `json:"provider_type"`
	ChannelType        string         `json:"channel_type"`
	InteractionMode    string         `json:"interaction_mode"`
	PaymentEnvironment string         `json:"payment_environment"`
	FeeRate            string         `json:"fee_rate,omitempty"`
	FixedFee           string         `json:"fixed_fee,omitempty"`
	MinAmount          string         `json:"min_amount,omitempty"`
	MaxAmount          string         `json:"max_amount,omitempty"`
	HideAmountOutRange bool           `json:"hide_amount_out_range,omitempty"`
	PaymentRoles       []string       `json:"payment_roles,omitempty"`
	PaymentTypes       []string       `json:"payment_types,omitempty"`
	ConfigJSON         map[string]any `json:"config_json"`
	IsActive           *bool          `json:"is_active,omitempty"`
	SortOrder          int            `json:"sort_order,omitempty"`
}

type UpdateAcquiringChannelStatusInput struct {
	IsActive bool `json:"is_active"`
}

type MailHealthResult struct {
	Source      string    `json:"source"`
	CollectedAt time.Time `json:"collected_at"`
	Freshness   string    `json:"freshness"`
	Enabled     bool      `json:"enabled"`
	Configured  bool      `json:"configured"`
	Host        string    `json:"host,omitempty"`
	Port        int       `json:"port,omitempty"`
	From        string    `json:"from,omitempty"`
	Status      string    `json:"status"`
}

type AcquiringStore interface {
	ListChannels() ([]AcquiringChannel, int64, error)
	UpsertChannel(input UpsertAcquiringChannelInput) (*AcquiringChannel, error)
	UpdateChannelStatus(id uint, active bool) (*AcquiringChannel, error)
	MailHealth() (*MailHealthResult, error)
}

type UpstreamConnection struct {
	ID                 uint       `json:"id"`
	Name               string     `json:"name"`
	BaseURL            string     `json:"base_url"`
	ApiKey             string     `json:"api_key"`
	Protocol           string     `json:"protocol"`
	CallbackURL        string     `json:"callback_url,omitempty"`
	Status             string     `json:"status"`
	LastPingAt         *time.Time `json:"last_ping_at,omitempty"`
	LastPingOK         bool       `json:"last_ping_ok"`
	ExchangeRate       string     `json:"exchange_rate"`
	PriceMarkupPercent string     `json:"price_markup_percent"`
	PriceRoundingMode  string     `json:"price_rounding_mode"`
	AutoSyncPrice      bool       `json:"auto_sync_price"`
	PingSiteName       string     `json:"ping_site_name,omitempty"`
	PingError          string     `json:"ping_error,omitempty"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type UpstreamConnectionsResult struct {
	Source      string               `json:"source"`
	CollectedAt time.Time            `json:"collected_at"`
	Freshness   string               `json:"freshness"`
	Connections []UpstreamConnection `json:"connections"`
	Total       int64                `json:"total"`
}

type SaveUpstreamConnectionInput struct {
	ID                 uint     `json:"id,omitempty"`
	Name               string   `json:"name"`
	BaseURL            string   `json:"base_url"`
	ApiKey             string   `json:"api_key"`
	ApiSecret          string   `json:"api_secret,omitempty"`
	Protocol           string   `json:"protocol,omitempty"`
	CallbackURL        string   `json:"callback_url,omitempty"`
	ExchangeRate       *float64 `json:"exchange_rate,omitempty"`
	PriceMarkupPercent *float64 `json:"price_markup_percent,omitempty"`
	PriceRoundingMode  *string  `json:"price_rounding_mode,omitempty"`
	AutoSyncPrice      *bool    `json:"auto_sync_price,omitempty"`
	Ping               *bool    `json:"ping,omitempty"`
}

type UpstreamCatalogCategory struct {
	ID        uint           `json:"id"`
	ParentID  uint           `json:"parent_id"`
	Slug      string         `json:"slug"`
	Name      map[string]any `json:"name"`
	SortOrder int            `json:"sort_order"`
}

type UpstreamCatalogProduct struct {
	ID                      uint           `json:"id"`
	CategoryID              uint           `json:"category_id"`
	Slug                    string         `json:"slug"`
	Title                   map[string]any `json:"title"`
	PriceAmount             string         `json:"price_amount"`
	Currency                string         `json:"currency"`
	IsActive                bool           `json:"is_active"`
	SKUCount                int            `json:"sku_count"`
	SKUCodes                []string       `json:"sku_codes,omitempty"`
	RequiresReservation     bool           `json:"requires_reservation"`
	EntitlementContractHash string         `json:"entitlement_contract_hash,omitempty"`
	Mapped                  bool           `json:"mapped"`
	MappingID               uint           `json:"mapping_id,omitempty"`
	LocalProductID          uint           `json:"local_product_id,omitempty"`
}

type UpstreamCatalogResult struct {
	Source              string                    `json:"source"`
	CollectedAt         time.Time                 `json:"collected_at"`
	Freshness           string                    `json:"freshness"`
	ConnectionID        uint                      `json:"connection_id"`
	CategoriesSupported bool                      `json:"categories_supported"`
	Categories          []UpstreamCatalogCategory `json:"categories"`
	Products            []UpstreamCatalogProduct  `json:"products"`
	Total               int                       `json:"total"`
}

type ImportUpstreamProductsInput struct {
	UpstreamProductIDs []uint `json:"upstream_product_ids,omitempty"`
	LocalCategoryID    uint   `json:"local_category_id,omitempty"`
	AutoCreateCategory *bool  `json:"auto_create_category,omitempty"`
	SyncStockOnly      bool   `json:"sync_stock_only,omitempty"`
	SKUCode            string `json:"sku_code,omitempty"`
}

type ImportUpstreamProductsFailure struct {
	UpstreamProductID uint   `json:"upstream_product_id"`
	Error             string `json:"error"`
}

type ImportUpstreamProductsResult struct {
	Source        string                          `json:"source"`
	CollectedAt   time.Time                       `json:"collected_at"`
	ConnectionID  uint                            `json:"connection_id"`
	Requested     int                             `json:"requested"`
	ImportedCount int                             `json:"imported_count"`
	SyncedCount   int                             `json:"synced_count"`
	SkippedCount  int                             `json:"skipped_count,omitempty"`
	SkippedReason string                          `json:"skipped_reason,omitempty"`
	Failures      []ImportUpstreamProductsFailure `json:"failures,omitempty"`
}

type UpdateUpstreamConnectionStatusInput struct {
	Status string `json:"status"`
}

type UpstreamStore interface {
	ListConnections() ([]UpstreamConnection, int64, error)
	SaveConnection(input SaveUpstreamConnectionInput) (*UpstreamConnection, error)
	UpdateConnectionStatus(id uint, status string) (*UpstreamConnection, error)
	DeleteConnection(id uint) error
	FetchCatalog(connectionID uint) (*UpstreamCatalogResult, error)
	ImportOrSync(connectionID uint, input ImportUpstreamProductsInput) (*ImportUpstreamProductsResult, error)
}

type RefundOrderInput struct {
	OperationID        string `json:"operation_id"`
	Mode               string `json:"mode,omitempty"`
	Amount             string `json:"amount"`
	Channel            string `json:"channel,omitempty"`
	ExternalTxRef      string `json:"external_tx_ref,omitempty"`
	Remark             string `json:"remark,omitempty"`
	PaymentFeeRefunded bool   `json:"payment_fee_refunded,omitempty"`
}

type RefundOrderResult struct {
	Source               string     `json:"source"`
	CollectedAt          time.Time  `json:"collected_at"`
	OrderID              uint       `json:"order_id"`
	OrderNo              string     `json:"order_no"`
	Status               string     `json:"status"`
	TotalAmount          string     `json:"total_amount"`
	RefundedAmount       string     `json:"refunded_amount"`
	Currency             string     `json:"currency"`
	RefundRecordID       uint       `json:"refund_record_id"`
	RefundType           string     `json:"refund_type"`
	SettlementStatus     string     `json:"settlement_status"`
	OperationID          string     `json:"operation_id"`
	BalanceSyncStatus    string     `json:"balance_sync_status,omitempty"`
	BalanceSyncOpID      string     `json:"balance_sync_op_id,omitempty"`
	BalanceSyncError     string     `json:"balance_sync_error,omitempty"`
	BalanceSyncUpdatedAt *time.Time `json:"balance_sync_updated_at,omitempty"`
}

type OrderRefundStore interface {
	RefundOrder(orderNo string, input RefundOrderInput) (*RefundOrderResult, error)
}

type SLORevokeUserInput struct {
	MerchantCustomerID string `json:"merchant_customer_id"`
	Reason             string `json:"reason,omitempty"`
}

type SLORevokeUserResult struct {
	Revoked bool `json:"revoked"`
	UserID  uint `json:"user_id,omitempty"`
}

type UserSessionRevoker interface {
	RevokeUserSessions(userID uint) (bool, error)
}

type SLORevokeHandler interface {
	RevokeUserSLO(input SLORevokeUserInput) (*SLORevokeUserResult, error)
}

type UniversalGiftCardClaimStatus struct {
	Source        string     `json:"source"`
	CollectedAt   time.Time  `json:"collected_at"`
	ClaimID       string     `json:"claim_id"`
	Found         bool       `json:"found"`
	UserID        uint       `json:"user_id,omitempty"`
	TransactionID uint       `json:"transaction_id,omitempty"`
	Amount        string     `json:"amount,omitempty"`
	Currency      string     `json:"currency,omitempty"`
	TxnRef        string     `json:"txn_ref,omitempty"`
	CreditDisplay string     `json:"credit_display,omitempty"`
	CreatedAt     *time.Time `json:"created_at,omitempty"`
}

type UniversalGiftCardClaimStore interface {
	LookupUniversalGiftCardClaim(claimID string) (*UniversalGiftCardClaimStatus, error)
}

type UniversalGiftCardClaimLookup interface {
	LookupUniversalGiftCardClaim(claimID string) (*UniversalGiftCardClaimStatus, error)
}

type ManualConfirmPaymentInput struct {
	PaymentID   uint   `json:"payment_id,omitempty"`
	ProviderRef string `json:"provider_ref,omitempty"`
	Remark      string `json:"remark,omitempty"`
}

type ManualConfirmPaymentResult struct {
	Source        string    `json:"source"`
	CollectedAt   time.Time `json:"collected_at"`
	OrderNo       string    `json:"order_no"`
	PaymentID     uint      `json:"payment_id"`
	PaymentStatus string    `json:"payment_status"`
	OrderStatus   string    `json:"order_status"`
}

type ManualFulfillOrderInput struct {
	TargetOrderID uint   `json:"target_order_id,omitempty"`
	Payload       string `json:"payload"`
}

type ManualFulfillOrderResult struct {
	Source        string    `json:"source"`
	CollectedAt   time.Time `json:"collected_at"`
	OrderNo       string    `json:"order_no"`
	TargetOrderID uint      `json:"target_order_id"`
	FulfillmentID uint      `json:"fulfillment_id"`
	Status        string    `json:"status"`
}

type UpdateOrderStatusInput struct {
	Status string `json:"status"`
}

type UpdateOrderStatusResult struct {
	Source      string    `json:"source"`
	CollectedAt time.Time `json:"collected_at"`
	OrderNo     string    `json:"order_no"`
	Status      string    `json:"status"`
}

type RetryProcurementResult struct {
	Source        string    `json:"source"`
	CollectedAt   time.Time `json:"collected_at"`
	ProcurementID uint      `json:"procurement_id"`
	Status        string    `json:"status"`
}

type OrderBuyerInfo struct {
	Email       string
	DisplayName string
}

type OrderEnricher interface {
	LookupBuyers(userIDs []uint) (map[uint]OrderBuyerInfo, error)
}

type OrderActionStore interface {
	ManualConfirmPayment(orderNo string, input ManualConfirmPaymentInput) (*ManualConfirmPaymentResult, error)
	ManualFulfillOrder(orderNo string, input ManualFulfillOrderInput) (*ManualFulfillOrderResult, error)
	UpdateOrderStatus(orderNo string, input UpdateOrderStatusInput) (*UpdateOrderStatusResult, error)
	RetryProcurement(id uint) (*RetryProcurementResult, error)
}

type StorefrontUser struct {
	ID             uint       `json:"id"`
	Email          string     `json:"email"`
	DisplayName    string     `json:"display_name"`
	Locale         string     `json:"locale"`
	Status         string     `json:"status"`
	MemberLevelID  uint       `json:"member_level_id"`
	WalletBalance  string     `json:"wallet_balance"`
	WalletCurrency string     `json:"wallet_currency"`
	TotalRecharged string     `json:"total_recharged"`
	TotalSpent     string     `json:"total_spent"`
	AdminNote      string     `json:"admin_note,omitempty"`
	TOTPEnabled    bool       `json:"totp_enabled"`
	EmailVerified  bool       `json:"email_verified"`
	OAuthProviders []string   `json:"oauth_providers,omitempty"`
	LastLoginAt    *time.Time `json:"last_login_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type UsersResult struct {
	Source      string           `json:"source"`
	CollectedAt time.Time        `json:"collected_at"`
	Freshness   string           `json:"freshness"`
	Users       []StorefrontUser `json:"users"`
	Total       int64            `json:"total"`
	Page        int              `json:"page"`
	PageSize    int              `json:"page_size"`
}

type CreateUserInput struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	DisplayName   string `json:"display_name,omitempty"`
	Locale        string `json:"locale,omitempty"`
	Status        string `json:"status,omitempty"`
	MemberLevelID uint   `json:"member_level_id,omitempty"`
	EmailVerified *bool  `json:"email_verified,omitempty"`
	AdminNote     string `json:"admin_note,omitempty"`
}

type UpdateUserInput struct {
	Email         *string `json:"email,omitempty"`
	Password      *string `json:"password,omitempty"`
	DisplayName   *string `json:"display_name,omitempty"`
	Locale        *string `json:"locale,omitempty"`
	Status        *string `json:"status,omitempty"`
	MemberLevelID *uint   `json:"member_level_id,omitempty"`
	EmailVerified *bool   `json:"email_verified,omitempty"`
	AdminNote     *string `json:"admin_note,omitempty"`
}

type AdjustUserWalletInput struct {
	Operation string `json:"operation"`
	Amount    string `json:"amount"`
	Currency  string `json:"currency,omitempty"`
	Remark    string `json:"remark,omitempty"`
}

type WalletAdjustmentResult struct {
	Source        string    `json:"source"`
	CollectedAt   time.Time `json:"collected_at"`
	UserID        uint      `json:"user_id"`
	Balance       string    `json:"balance"`
	Currency      string    `json:"currency"`
	TransactionID uint      `json:"transaction_id"`
	Direction     string    `json:"direction"`
	Amount        string    `json:"amount"`
}

type UserStore interface {
	ListUsers(page, pageSize int, keyword, status string) (*UsersResult, error)
	CreateUser(input CreateUserInput) (*StorefrontUser, error)
	UpdateUser(id uint, input UpdateUserInput) (*StorefrontUser, error)
	DeleteUser(id uint) error
	AdjustUserWallet(id uint, input AdjustUserWalletInput) (*WalletAdjustmentResult, error)
	ClearUserTOTP(id uint) error
	UnbindUserOAuth(id uint, provider string) error
}

type RadarSubscribersFilter struct {
	Page          int
	PageSize      int
	Keyword       string
	Channel       string
	DeliveryState string
	Excluded      string
	Scope         string
}

type RadarSubscribersSummary struct {
	SubscribedMembers int            `json:"subscribed_members"`
	TotalMembers      int            `json:"total_members"`
	ChannelCounts     map[string]int `json:"channel_counts"`
	TotalSent         int            `json:"total_sent"`
	TotalFailed       int            `json:"total_failed"`
	FailedMembers     int            `json:"failed_members"`
	ExcludedMembers   int            `json:"excluded_members"`
}

type RadarSubscriberItem struct {
	UserID             uint       `json:"user_id"`
	Email              string     `json:"email"`
	DisplayName        string     `json:"display_name"`
	UserStatus         string     `json:"user_status"`
	Channels           []string   `json:"channels"`
	SubscribedCount    int        `json:"subscribed_count"`
	SentCount          int        `json:"sent_count"`
	FailedCount        int        `json:"failed_count"`
	TotalDeliveries    int        `json:"total_deliveries"`
	HasFailure         bool       `json:"has_failure"`
	LastDeliveryStatus string     `json:"last_delivery_status,omitempty"`
	LastDeliveryAt     *time.Time `json:"last_delivery_at,omitempty"`
	LastError          string     `json:"last_error,omitempty"`
	LastFailedAt       *time.Time `json:"last_failed_at,omitempty"`
	Excluded           bool       `json:"excluded"`
	ExcludeReason      string     `json:"exclude_reason,omitempty"`
	ExcludedBy         string     `json:"excluded_by,omitempty"`
	ExcludedAt         *time.Time `json:"excluded_at,omitempty"`
	UpdatedAt          *time.Time `json:"updated_at,omitempty"`
}

type RadarSubscribersResult struct {
	Source      string                  `json:"source"`
	CollectedAt time.Time               `json:"collected_at"`
	Freshness   string                  `json:"freshness"`
	Summary     RadarSubscribersSummary `json:"summary"`
	Items       []RadarSubscriberItem   `json:"items"`
	Total       int64                   `json:"total"`
	Page        int                     `json:"page"`
	PageSize    int                     `json:"page_size"`
}

type RadarDeliveryItem struct {
	ID        uint       `json:"id"`
	Channel   string     `json:"channel"`
	RecordID  string     `json:"record_id"`
	UserID    uint       `json:"user_id"`
	Email     string     `json:"email"`
	Status    string     `json:"status"`
	Attempts  int        `json:"attempts"`
	LastError string     `json:"last_error,omitempty"`
	SentAt    *time.Time `json:"sent_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type RadarDeliveriesResult struct {
	Source      string              `json:"source"`
	CollectedAt time.Time           `json:"collected_at"`
	Freshness   string              `json:"freshness"`
	UserID      uint                `json:"user_id"`
	Email       string              `json:"email"`
	DisplayName string              `json:"display_name"`
	Items       []RadarDeliveryItem `json:"items"`
	Total       int                 `json:"total"`
}

type UpdateRadarExclusionInput struct {
	Excluded bool   `json:"excluded"`
	Reason   string `json:"reason,omitempty"`
	Operator string `json:"operator,omitempty"`
}

type RadarExclusionResult struct {
	Source      string    `json:"source"`
	CollectedAt time.Time `json:"collected_at"`
	UserID      uint      `json:"user_id"`
	Excluded    bool      `json:"excluded"`
	Reason      string    `json:"reason,omitempty"`
	UpdatedBy   string    `json:"updated_by,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RadarSubscriptionStore interface {
	RadarSubscribers(filter RadarSubscribersFilter) (*RadarSubscribersResult, error)
	RadarSubscriberDeliveries(userID uint, limit int) (*RadarDeliveriesResult, error)
	SetRadarSubscriberExclusion(userID uint, input UpdateRadarExclusionInput) (*RadarExclusionResult, error)
}

type RadarSubscriptionHandler interface {
	RadarSubscribers(filter RadarSubscribersFilter) (*RadarSubscribersResult, error)
	RadarSubscriberDeliveries(userID uint, limit int) (*RadarDeliveriesResult, error)
	SetRadarSubscriberExclusion(userID uint, input UpdateRadarExclusionInput) (*RadarExclusionResult, error)
}

type Reader interface {
	Orders(page, pageSize int) (*OrdersResult, error)
	Overview() (*Overview, error)
}

type Bridge interface {
	Reader
	Categories() (*CategoriesResult, error)
	CreateCategory(input CreateCategoryInput) (*Category, error)
	UpdateCategory(id string, input UpdateCategoryInput) (*Category, error)
	DeleteCategory(id string) error
	Products(page, pageSize int, categoryID, search string) (*ProductsResult, error)
	CreateProduct(input CreateProductInput) (*Product, error)
	UpdateProduct(id string, input UpdateProductInput) (*Product, error)
	DeleteProduct(id string) error
	BatchDeleteProducts(input BatchDeleteProductsInput) (*BatchDeleteResult, error)
	CreateSKU(productID uint, input CreateSKUInput) (*ProductSKU, error)
	UpdateSKU(id uint, input UpdateSKUInput) (*ProductSKU, error)
	DeleteSKU(id uint) error
	Coupons(page, pageSize int, code string, isActive *bool) (*CouponsResult, error)
	CreateCoupon(input CreateCouponInput) (*Coupon, error)
	UpdateCouponStatus(id uint, active bool) (*Coupon, error)
	GiftCards(page, pageSize int, code, status, batchNo string, redeemedUserID uint) (*GiftCardsResult, error)
	GenerateGiftCards(input GenerateGiftCardsInput) (*GenerateGiftCardsResult, error)
	Pricing() (*PricingSettings, error)
	UpdatePricing(input UpdatePricingInput) (*PricingSettings, error)
	AcquiringChannels() (*AcquiringChannelsResult, error)
	UpsertAcquiringChannel(input UpsertAcquiringChannelInput) (*AcquiringChannel, error)
	UpdateAcquiringChannelStatus(id uint, active bool) (*AcquiringChannel, error)
	MailHealth() (*MailHealthResult, error)
	UpstreamConnections() (*UpstreamConnectionsResult, error)
	SaveUpstreamConnection(input SaveUpstreamConnectionInput) (*UpstreamConnection, error)
	UpdateUpstreamConnectionStatus(id uint, status string) (*UpstreamConnection, error)
	DeleteUpstreamConnection(id uint) error
	UpstreamCatalog(connectionID uint) (*UpstreamCatalogResult, error)
	ImportUpstreamProducts(connectionID uint, input ImportUpstreamProductsInput) (*ImportUpstreamProductsResult, error)
	RefundOrder(orderNo string, input RefundOrderInput) (*RefundOrderResult, error)
	OrdersFiltered(filter OrdersFilter) (*OrdersResult, error)
	ManualConfirmPayment(orderNo string, input ManualConfirmPaymentInput) (*ManualConfirmPaymentResult, error)
	ManualFulfillOrder(orderNo string, input ManualFulfillOrderInput) (*ManualFulfillOrderResult, error)
	UpdateOrderStatus(orderNo string, input UpdateOrderStatusInput) (*UpdateOrderStatusResult, error)
	RetryProcurement(id uint) (*RetryProcurementResult, error)
	Users(page, pageSize int, keyword, status string) (*UsersResult, error)
	CreateUser(input CreateUserInput) (*StorefrontUser, error)
	UpdateUser(id uint, input UpdateUserInput) (*StorefrontUser, error)
	DeleteUser(id uint) error
	AdjustUserWallet(id uint, input AdjustUserWalletInput) (*WalletAdjustmentResult, error)
	ClearUserTOTP(id uint) error
	UnbindUserOAuth(id uint, provider string) error
}
