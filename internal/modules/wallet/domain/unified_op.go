package domain

import (
	"time"

	"github.com/dujiao-next/internal/shared/money"
)

const (
	UnifiedOpDirectionDebit  = "debit"
	UnifiedOpDirectionCredit = "credit"

	UnifiedOpStatusPending         = "pending"
	UnifiedOpStatusDebited         = "debited"
	UnifiedOpStatusApplied         = "applied"
	UnifiedOpStatusCancelled       = "cancelled"
	UnifiedOpStatusRollbackPending = "rollback_pending"
	UnifiedOpStatusRolledBack      = "rolled_back"
	UnifiedOpStatusCreditPending   = "credit_pending"
	UnifiedOpStatusCredited        = "credited"
)

// UnifiedOp 持久化跨系统统一余额（Storefront <-> NewAPI）的扣款占位、入账 Outbox 与冲正状态，
// 保证任意进程崩溃、网络超时或并发重试都能安全恢复，且永不复用已被冲正的扣款操作号。
type UnifiedOp struct {
	ID          uint         `gorm:"primarykey" json:"id"`
	OpID        string       `gorm:"type:varchar(128);uniqueIndex;not null" json:"op_id"`
	Direction   string       `gorm:"type:varchar(16);index;not null" json:"direction"`
	Status      string       `gorm:"type:varchar(32);index;not null" json:"status"`
	UserID      uint         `gorm:"index;not null" json:"user_id"`
	OrderID     *uint        `gorm:"index" json:"order_id,omitempty"`
	Amount      money.Amount `gorm:"type:decimal(20,2);not null" json:"amount"`
	Currency    string       `gorm:"type:varchar(16);not null;default:'CNY'" json:"currency"`
	Remark      string       `gorm:"type:varchar(255)" json:"remark"`
	LastError   string       `gorm:"type:varchar(500)" json:"last_error,omitempty"`
	Attempts    int          `gorm:"not null;default:0" json:"attempts"`
	NextRetryAt *time.Time   `gorm:"index" json:"next_retry_at,omitempty"`
	CreatedAt   time.Time    `gorm:"index" json:"created_at"`
	UpdatedAt   time.Time    `gorm:"index" json:"updated_at"`
}

func (UnifiedOp) TableName() string {
	return "wallet_unified_ops"
}
