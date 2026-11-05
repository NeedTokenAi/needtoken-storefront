package contract

import (
	"context"
	"encoding/json"
	"time"

	categoryapp "github.com/dujiao-next/internal/modules/catalog/category/application"
	categorydomain "github.com/dujiao-next/internal/modules/catalog/category/domain"
	settingsintegration "github.com/dujiao-next/internal/modules/settings/schema/integration"
	siteconnectiondomain "github.com/dujiao-next/internal/modules/siteconnection/domain"
	"github.com/dujiao-next/internal/upstream"
)

// ConnectionProvider 隔离连接读取与上游协议适配器构造。
type ConnectionProvider interface {
	GetByID(id uint) (*siteconnectiondomain.Connection, error)
	GetAdapter(conn *siteconnectiondomain.Connection) (upstream.Adapter, error)
}

// MediaRecorder 记录导入后落盘的上游媒体。
type MediaRecorder interface {
	RecordLocalFile(ctx context.Context, localPath, scene string)
}

// CategoryCreator 自动创建导入商品所需分类。
type CategoryCreator interface {
	Create(input categoryapp.UpsertInput) (*categorydomain.Category, error)
}

// SettingsProvider 提供上游同步动态配置。
type SettingsProvider interface {
	GetUpstreamSyncConfig(fallbackInterval string) (settingsintegration.UpstreamSyncConfig, error)
	GetUpstreamSyncInterval(fallbackInterval string) (time.Duration, error)
}

// PlanCheckoutRoute is resolved from the active merchant SKU mapping, then frozen
// with the order. Payment rechecks this route rather than a mutable local mapping.
type PlanCheckoutRoute struct {
	ConnectionID  uint
	UpstreamSKUID uint
}

type PlanCheckoutRequest struct {
	Route                   PlanCheckoutRoute
	UserID                  uint
	Quantity                int
	EntitlementContract     json.RawMessage
	EntitlementContractHash string
}

// PlanCheckoutChecker must run outside a database transaction. It is read-only,
// requires a merchant member identity and never uses a submitted contact email.
type PlanCheckoutChecker interface {
	CheckPlanCheckout(request PlanCheckoutRequest) error
}

type PlanCheckoutPreparer interface {
	PreparePlanCheckout(localSKUID, userID uint, quantity int, contract json.RawMessage, hash string) (PlanCheckoutRoute, error)
}
