package contract

import (
	"time"

	walletdomain "github.com/dujiao-next/internal/modules/wallet/domain"

	userdomain "github.com/dujiao-next/internal/modules/identity/user/domain"

	giftcarddomain "github.com/dujiao-next/internal/modules/giftcard/domain"
	"github.com/dujiao-next/internal/shared/money"
)

// ListFilter 礼品卡仓储列表筛选。
type ListFilter struct {
	Code           string
	Status         string
	BatchNo        string
	RedeemedUserID uint
	CreatedFrom    *time.Time
	CreatedTo      *time.Time
	RedeemedFrom   *time.Time
	RedeemedTo     *time.Time
	ExpiresFrom    *time.Time
	ExpiresTo      *time.Time
	Page           int
	PageSize       int
}

// Repository 是礼品卡管理用例的数据端口。
type Repository interface {
	CreateBatch(batch *giftcarddomain.GiftCardBatch, cards []giftcarddomain.GiftCard) error
	GetByID(id uint) (*giftcarddomain.GiftCard, error)
	List(filter ListFilter) ([]giftcarddomain.GiftCard, int64, error)
	ListByIDs(ids []uint) ([]giftcarddomain.GiftCard, error)
	Update(card *giftcarddomain.GiftCard) error
	Delete(id uint) error
	BatchUpdateStatus(ids []uint, status string, updatedAt time.Time) (int64, error)
	WithinTransaction(fn func(repo Repository) error) error
}

// WalletCreditInput 描述礼品卡兑换需要的钱包入账事实。
type WalletCreditInput struct {
	UserID    uint
	Amount    money.Amount
	Currency  string
	TxnType   string
	Reference string
	Remark    string
	OrderID   *uint
}

// RedeemTransaction 是礼品卡与钱包共享事务内的最小能力集合。
type RedeemTransaction interface {
	GetByCodeForUpdate(code string) (*giftcarddomain.GiftCard, error)
	UpdateCard(card *giftcarddomain.GiftCard) error
	CreditWallet(input WalletCreditInput) (*walletdomain.Account, *walletdomain.Transaction, error)
}

// RedeemTransactionRunner 保证礼品卡状态与钱包入账原子提交或回滚。
type RedeemTransactionRunner interface {
	WithinRedeemTransaction(fn func(tx RedeemTransaction) error) error
}

// UserDirectory 是兑换用户解析端口。
type UserDirectory interface {
	ListByIDs(ids []uint) ([]userdomain.User, error)
}

// CurrencyProvider 是站点币种读取端口。
type CurrencyProvider interface {
	SiteCurrency() string
}

// UniversalGiftCardClaim 描述从跨系统权威源（Glue）原子预锁定的通用礼品卡凭据。
type UniversalGiftCardClaim struct {
	ClaimID      string
	Code         string
	Name         string
	BatchNo      string
	CreditAmount money.Amount
	Currency     string
	FaceUSDCents int64
	ExpiresAt    *time.Time
}

// UniversalGiftCardRedeemer 提供双端通用礼品卡（一卡两用防超售）的跨系统二阶段核销端口。
// 所有调用必须在本地数据库事务外执行，避免占用 SQLite 写锁或长事务。
type UniversalGiftCardRedeemer interface {
	Claim(code string, userID uint, siteCurrency string) (*UniversalGiftCardClaim, error)
	Confirm(code, claimID, txnRef, creditDisplay string) error
	Release(code, claimID, reason string) error
}
