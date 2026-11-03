package domain

import (
	"time"

	"github.com/dujiao-next/internal/shared/jsonmap"
)

// CheckoutReservation is a durable, content-bound request to the authoritative
// upstream capacity book. It is not a second inventory balance. An unresolved
// request always reuses its original operation/snapshot; it cannot allocate
// replacement stock after a payment or a timeout.
type CheckoutReservation struct {
	OrderItemID        uint         `gorm:"primarykey;autoIncrement:false" json:"order_item_id"`
	RootOrderID        uint         `gorm:"index;not null" json:"root_order_id"`
	OrderID            uint         `gorm:"index;not null" json:"order_id"`
	ConnectionID       uint         `gorm:"not null" json:"connection_id"`
	UpstreamSKUID      uint         `gorm:"not null" json:"upstream_sku_id"`
	UpstreamIdentity   string       `gorm:"size:64" json:"upstream_identity,omitempty"`
	RequirementHash    string       `gorm:"size:64;not null" json:"requirement_hash"`
	OrderRef           string       `gorm:"size:96;uniqueIndex;not null" json:"order_ref"`
	ReserveOperationID string       `gorm:"size:96;uniqueIndex;not null" json:"reserve_operation_id"`
	ReservePayloadHash string       `gorm:"size:64;not null" json:"reserve_payload_hash"`
	ReserveRequestJSON jsonmap.JSON `gorm:"type:json;not null" json:"-"`
	ConfirmRequestJSON jsonmap.JSON `gorm:"type:json" json:"-"`
	HoldID             string       `gorm:"size:128" json:"hold_id,omitempty"`
	HoldVersion        string       `gorm:"size:20" json:"hold_version,omitempty"`
	State              string       `gorm:"size:32;index;not null" json:"state"`
	BookStatus         string       `gorm:"size:32" json:"book_status,omitempty"`
	ReadyForPayment    bool         `gorm:"not null;default:false" json:"ready_for_payment"`
	ExpiresAt          time.Time    `gorm:"index;not null" json:"expires_at"`
	AttentionCode      string       `gorm:"size:64" json:"attention_code,omitempty"`
	CreatedAt          time.Time    `json:"created_at"`
	UpdatedAt          time.Time    `json:"updated_at"`
}

func (CheckoutReservation) TableName() string { return "checkout_reservations" }
