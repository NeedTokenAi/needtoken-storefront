package domain

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	siteconnectiondomain "github.com/dujiao-next/internal/modules/siteconnection/domain"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/dujiao-next/internal/shared/money"
)

const PayloadPreviewMaxLines = 100

// Order 是采购上下文的聚合根。
type Order struct {
	ID                       uint         `gorm:"primarykey" json:"id"`
	ConnectionID             uint         `gorm:"index;not null" json:"connection_id"`
	LocalOrderID             uint         `gorm:"uniqueIndex;not null" json:"local_order_id"`
	LocalOrderNo             string       `gorm:"type:varchar(64);index" json:"local_order_no"`
	UpstreamOrderID          uint         `json:"-"`
	UpstreamOrderNo          string       `gorm:"type:varchar(64);index" json:"upstream_order_no,omitempty"`
	Status                   string       `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
	UpstreamAmount           money.Amount `gorm:"type:decimal(20,2);not null;default:0" json:"upstream_amount"`
	UpstreamCurrency         string       `gorm:"type:varchar(10);not null;default:''" json:"upstream_currency"`
	LocalSellAmount          money.Amount `gorm:"type:decimal(20,2);not null;default:0" json:"local_sell_amount"`
	Currency                 string       `gorm:"type:varchar(10);not null" json:"currency"`
	ErrorMessage             string       `gorm:"type:text" json:"error_message,omitempty"`
	RetryCount               int          `gorm:"not null;default:0" json:"retry_count"`
	NextRetryAt              *time.Time   `gorm:"index" json:"next_retry_at,omitempty"`
	UpstreamPayload          string       `gorm:"type:text" json:"upstream_payload,omitempty"`
	UpstreamPayloadLineCount int          `gorm:"-" json:"upstream_payload_line_count"`
	TraceID                  string       `gorm:"type:varchar(64);index" json:"trace_id"`
	CreatedAt                time.Time    `gorm:"index" json:"created_at"`
	UpdatedAt                time.Time    `gorm:"index" json:"updated_at"`
	DeletedAt                *time.Time   `gorm:"index" json:"-"`

	Connection             *siteconnectiondomain.Connection `gorm:"foreignKey:ConnectionID" json:"connection,omitempty"`
	LocalOrder             *LocalOrder                      `gorm:"-" json:"local_order,omitempty"`
	LocalOrderReference    *LocalOrderReference             `gorm:"foreignKey:LocalOrderID;references:ID;constraint:fk_procurement_orders_local_order,OnUpdate:NO ACTION,OnDelete:NO ACTION" json:"-"`
	ParentOrderNo          string                           `gorm:"-" json:"parent_order_no,omitempty"`
	UpstreamRefundRecords  []jsonmap.JSON                   `gorm:"-" json:"upstream_refund_records,omitempty"`
	UpstreamRefundedAmount string                           `gorm:"-" json:"upstream_refunded_amount,omitempty"`
}

// LocalOrderReference only carries cross-context relationship metadata for
// schema migration. Procurement application code consumes LocalOrder snapshots.
type LocalOrderReference struct {
	ID uint `gorm:"primarykey"`
}

func (LocalOrderReference) TableName() string { return "orders" }

func (Order) TableName() string { return "procurement_orders" }

func (o *Order) TruncateUpstreamPayload(maxLines int) {
	if o == nil || o.UpstreamPayload == "" {
		return
	}
	lines := strings.Split(o.UpstreamPayload, "\n")
	o.UpstreamPayloadLineCount = len(lines)
	if len(lines) > maxLines {
		o.UpstreamPayload = strings.Join(lines[:maxLines], "\n")
	}
}

// LocalOrder 是采购应用读取的订单最小快照，同时保持管理端所需的 JSON 形状。
type LocalOrder struct {
	ID             uint             `json:"id"`
	OrderNo        string           `json:"order_no"`
	ParentID       *uint            `json:"parent_id,omitempty"`
	UserID         uint             `json:"user_id,omitempty"`
	GuestEmail     string           `json:"guest_email,omitempty"`
	Status         string           `json:"status"`
	Currency       string           `json:"currency"`
	TotalAmount    money.Amount     `json:"total_amount"`
	RefundedAmount money.Amount     `json:"refunded_amount"`
	Items          []LocalOrderItem `json:"items,omitempty"`
	Children       []LocalOrder     `json:"children,omitempty"`
}

type LocalOrderItem struct {
	ProductID                uint         `json:"product_id"`
	SKUID                    uint         `json:"sku_id"`
	Title                    jsonmap.JSON `json:"title"`
	SKUSnapshot              jsonmap.JSON `json:"sku_snapshot"`
	CostPrice                money.Amount `json:"cost_price"`
	Quantity                 int          `json:"quantity"`
	TotalPrice               money.Amount `json:"total_price"`
	FulfillmentType          string       `json:"fulfillment_type"`
	ManualFormSubmissionJSON jsonmap.JSON `json:"manual_form_submission"`
}

// Exact signed entitlement JSON is stored as a string because the historical
// generic order snapshot decoder uses float64. Preserve legacy paid snapshots
// while new plan orders use their exact contract and frozen upstream route.
func FrozenPlanProcurement(snapshot jsonmap.JSON) (jsonmap.JSON, uint, uint, bool, error) {
	raw, present := snapshot["plan_checkout_entitlement_raw"]
	if !present {
		return nil, 0, 0, false, nil
	}
	text, ok := raw.(string)
	var contract jsonmap.ExactJSON
	if !ok || len(text) == 0 || len(text) > 128*1024 || contract.UnmarshalJSON([]byte(text)) != nil || contract["kind"] != "plan" {
		return nil, 0, 0, true, fmt.Errorf("invalid frozen plan entitlement")
	}
	connectionText, _ := snapshot["plan_checkout_connection_id"].(string)
	skuText, _ := snapshot["plan_checkout_upstream_sku_id"].(string)
	connectionID, connectionErr := strconv.ParseUint(connectionText, 10, 64)
	skuID, skuErr := strconv.ParseUint(skuText, 10, 64)
	if connectionErr != nil || skuErr != nil || connectionID == 0 || skuID == 0 || uint64(uint(connectionID)) != connectionID || uint64(uint(skuID)) != skuID {
		return nil, 0, 0, true, fmt.Errorf("invalid frozen plan route")
	}
	return jsonmap.JSON(contract), uint(connectionID), uint(skuID), true, nil
}
