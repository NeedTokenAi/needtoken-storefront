package contract

import "time"

type Payment struct {
	ID          uint       `json:"id"`
	Status      string     `json:"status"`
	Amount      string     `json:"amount"`
	Currency    string     `json:"currency"`
	ProviderRef string     `json:"provider_ref,omitempty"`
	PaidAt      *time.Time `json:"paid_at,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at"`
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
type Reader interface {
	Orders(page, pageSize int) (*OrdersResult, error)
	Overview() (*Overview, error)
}
