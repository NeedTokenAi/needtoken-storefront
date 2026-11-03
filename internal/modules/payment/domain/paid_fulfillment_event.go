package domain

import "time"

// PaidFulfillmentEvent durably requests procurement creation. OrderID is the
// business idempotency key; queue delivery is only an acceleration mechanism.
type PaidFulfillmentEvent struct {
	OrderID       uint       `gorm:"primarykey;autoIncrement:false" json:"order_id"`
	Attempts      int        `gorm:"not null;default:0" json:"attempts"`
	NextAttemptAt time.Time  `gorm:"index" json:"next_attempt_at"`
	LeaseID       string     `gorm:"size:64" json:"-"`
	LeaseUntil    *time.Time `gorm:"index" json:"lease_until,omitempty"`
	CompletedAt   *time.Time `gorm:"index" json:"completed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (PaidFulfillmentEvent) TableName() string { return "paid_fulfillment_events" }
