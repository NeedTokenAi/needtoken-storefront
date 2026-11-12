package migrations

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/dujiao-next/internal/constants"
	affiliatedomain "github.com/dujiao-next/internal/modules/affiliate/domain"
	apicredentialdomain "github.com/dujiao-next/internal/modules/apicredential/domain"
	auditlogdomain "github.com/dujiao-next/internal/modules/auditlog/domain"
	cardsecretdomain "github.com/dujiao-next/internal/modules/cardsecret/domain"
	cartdomain "github.com/dujiao-next/internal/modules/cart/domain"
	categorydomain "github.com/dujiao-next/internal/modules/catalog/category/domain"
	mappingdomain "github.com/dujiao-next/internal/modules/catalog/mapping/domain"
	productdomain "github.com/dujiao-next/internal/modules/catalog/product/domain"
	channelclientdomain "github.com/dujiao-next/internal/modules/channelclient/domain"
	contentdomain "github.com/dujiao-next/internal/modules/content/domain"
	coupondomain "github.com/dujiao-next/internal/modules/coupon/domain"
	downstreamcallbackdomain "github.com/dujiao-next/internal/modules/downstreamcallback/domain"
	fulfillmentdomain "github.com/dujiao-next/internal/modules/fulfillment/domain"
	giftcarddomain "github.com/dujiao-next/internal/modules/giftcard/domain"
	admindomain "github.com/dujiao-next/internal/modules/identity/admin/domain"
	emailverificationdomain "github.com/dujiao-next/internal/modules/identity/emailverification/domain"
	externalidentitydomain "github.com/dujiao-next/internal/modules/identity/externalidentity/domain"
	invitecodedomain "github.com/dujiao-next/internal/modules/identity/invitecode/domain"
	userdomain "github.com/dujiao-next/internal/modules/identity/user/domain"
	memberleveldomain "github.com/dujiao-next/internal/modules/memberlevel/domain"
	notificationdomain "github.com/dujiao-next/internal/modules/notification/domain"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	orderriskcontract "github.com/dujiao-next/internal/modules/orderrisk/contract"
	orderriskdomain "github.com/dujiao-next/internal/modules/orderrisk/domain"
	paymentdomain "github.com/dujiao-next/internal/modules/payment/domain"
	procurementdomain "github.com/dujiao-next/internal/modules/procurement/domain"
	promotiondomain "github.com/dujiao-next/internal/modules/promotion/domain"
	reconciliationdomain "github.com/dujiao-next/internal/modules/reconciliation/domain"
	resellerstore "github.com/dujiao-next/internal/modules/reseller/infrastructure/gormstore"
	settingsstore "github.com/dujiao-next/internal/modules/settings/infrastructure/gormstore"
	siteconnectiondomain "github.com/dujiao-next/internal/modules/siteconnection/domain"
	broadcastdomain "github.com/dujiao-next/internal/modules/telegram/broadcast/domain"
	walletdomain "github.com/dujiao-next/internal/modules/wallet/domain"
	"github.com/dujiao-next/internal/platform/database/gormdb"

	"gorm.io/gorm"
)

// AutoMigrate owns the application-wide schema registry and ordered data migrations.
func AutoMigrate() error {
	return runAutoMigrate(context.Background())
}

func runAutoMigrate(ctx context.Context) error {
	db := gormdb.DB
	if err := migrationStep(ctx, func() error {
		return db.AutoMigrate(
			&admindomain.Admin{},
			&userdomain.User{},
			&externalidentitydomain.Identity{},
			&invitecodedomain.InviteCode{},
			&affiliatedomain.Profile{},
			&affiliatedomain.Click{},
			&affiliatedomain.Commission{},
			&affiliatedomain.WithdrawRequest{},
			&walletdomain.Account{},
			&walletdomain.Transaction{},
			&walletdomain.RechargeOrder{},
			&auditlogdomain.UserLoginLog{},
			&auditlogdomain.AuthzAuditLog{},
			&notificationdomain.NotificationLog{},
			&auditlogdomain.AdminLoginLog{},
			&emailverificationdomain.Code{},
			&orderdomain.Order{},
			&orderdomain.OrderItem{},
			&orderdomain.OrderRefundRecord{},
			&orderriskdomain.LockKey{},
			&cartdomain.Item{},
			&paymentdomain.PaymentChannel{},
			&paymentdomain.Payment{},
			&paymentdomain.PaidFulfillmentEvent{},
			&paymentdomain.CheckoutReservation{},
			&cardsecretdomain.Secret{},
			&cardsecretdomain.Batch{},
			&giftcarddomain.GiftCard{},
			&giftcarddomain.GiftCardBatch{},
			&fulfillmentdomain.Fulfillment{},
			&coupondomain.Coupon{},
			&coupondomain.CouponUsage{},
			&promotiondomain.Promotion{},
			&categorydomain.Category{},
			&productdomain.Product{},
			&productdomain.ProductSKU{},
			&contentdomain.Post{},
			&contentdomain.PostProduct{},
			&contentdomain.PostCategory{},
			&contentdomain.Banner{},
			&settingsstore.SettingRecord{},
			&apicredentialdomain.ApiCredential{},
			&siteconnectiondomain.Connection{},
			&mappingdomain.Mapping{},
			&mappingdomain.SKUMapping{},
			&procurementdomain.Order{},
			&downstreamcallbackdomain.OrderRef{},
			&reconciliationdomain.Job{},
			&reconciliationdomain.Item{},
			&channelclientdomain.Client{},
			&broadcastdomain.Broadcast{},
			&memberleveldomain.MemberLevel{},
			&memberleveldomain.MemberLevelPrice{},
			&contentdomain.Media{},
		)
	}); err != nil {
		return err
	}

	if err := migrationStep(ctx, ensureUserOAuthIdentityUserProviderUniqueIndex); err != nil {
		return err
	}
	if err := migrationStep(ctx, func() error { return backfillPendingOrderRiskIPs(db) }); err != nil {
		return err
	}
	if err := migrationStep(ctx, func() error { return resellerstore.Migrate(db) }); err != nil {
		return err
	}
	if err := migrationStep(ctx, migrateCartSKUUniqueIndex); err != nil {
		return err
	}
	if err := migrationStep(ctx, ensureProductSKUMigration); err != nil {
		return err
	}
	if err := migrationStep(ctx, ensureManualStockRemainingMigration); err != nil {
		return err
	}
	if err := migrationStep(ctx, ensureCategoryParentMigration); err != nil {
		return err
	}
	if err := migrationStep(ctx, ensurePaymentProviderBepusdtRenameMigration); err != nil {
		return err
	}
	if err := migrationStep(ctx, ensurePaymentChannelBepusdtConfigMigration); err != nil {
		return err
	}
	if err := migrationStep(ctx, ensurePaymentFeePolicyMigration); err != nil {
		return err
	}
	if err := migrationStep(ctx, ensureOrderRefundPaymentFeeMigration); err != nil {
		return err
	}
	if err := migrationStep(ctx, ensureOrderItemOriginalPriceMigration); err != nil {
		return err
	}
	if err := migrationStep(ctx, ensureCartForeignKeyConstraints); err != nil {
		return err
	}
	if err := migrationStep(ctx, ensureProcurementOrderForeignKeyConstraint); err != nil {
		return err
	}
	if err := migrationStep(ctx, func() error {
		if db.Migrator().HasColumn(&productdomain.Product{}, "price_currency") {
			return db.Migrator().DropColumn(&productdomain.Product{}, "price_currency")
		}
		return nil
	}); err != nil {
		return err
	}
	return migrationLease(ctx)
}

// backfillPendingOrderRiskIPs 只迁移仍可能占用库存的在途订单，避免启动时扫描全部历史订单。
func backfillPendingOrderRiskIPs(db *gorm.DB) error {
	type orderIPRow struct {
		ID       uint
		ClientIP string
	}
	var rows []orderIPRow
	if err := db.Model(&orderdomain.Order{}).
		Select("id", "client_ip").
		Where("deleted_at IS NULL AND status = ? AND (risk_ip IS NULL OR risk_ip = '') AND client_ip <> ''", constants.OrderStatusPendingPayment).
		Find(&rows).Error; err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, row := range rows {
			riskIP := orderriskcontract.NormalizeRiskIP(row.ClientIP)
			if riskIP == "" {
				continue
			}
			if err := tx.Model(&orderdomain.Order{}).
				Where("id = ? AND (risk_ip IS NULL OR risk_ip = '')", row.ID).
				UpdateColumn("risk_ip", riskIP).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

var migrationDBMu sync.Mutex

func migrationLease(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("writer permit lost during migration: %w", err)
	}
	return nil
}

func migrationStep(ctx context.Context, fn func() error) error {
	if err := migrationLease(ctx); err != nil {
		return err
	}
	return fn()
}

func withMigrationDB(db *gorm.DB, fn func() error) error {
	migrationDBMu.Lock()
	defer migrationDBMu.Unlock()
	previous := gormdb.DB
	gormdb.DB = db
	defer func() { gormdb.DB = previous }()
	return fn()
}

// runUnderLease applies every step in one transaction. A lost permit returns
// before commit, so the next start can run the same steps again.
func runUnderLease(ctx context.Context, db *gorm.DB, steps ...func(context.Context, *gorm.DB) error) error {
	if ctx == nil || ctx.Err() != nil {
		return errors.New("cloudflare profile: writer permit lost before migrations; this lease does not stop the existing Cloud Run writer")
	}
	if db == nil {
		return errors.New("database is not initialized")
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tx = tx.WithContext(ctx)
		for _, step := range steps {
			if err := migrationLease(ctx); err != nil {
				return err
			}
			if err := step(ctx, tx); err != nil {
				return err
			}
		}
		return migrationLease(ctx)
	})
}

// AutoMigrateUnderLease is the Cloudflare startup path. The writer permit is
// already held. Losing it rolls this attempt back instead of leaving a
// half-applied migration.
func AutoMigrateUnderLease(ctx context.Context) error {
	return runUnderLease(ctx, gormdb.DB, func(ctx context.Context, tx *gorm.DB) error {
		return withMigrationDB(tx, func() error {
			return runAutoMigrate(ctx)
		})
	})
}
