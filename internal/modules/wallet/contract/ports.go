package contract

import (
	"context"
	"time"

	"github.com/dujiao-next/internal/shared/money"
	"github.com/shopspring/decimal"

	walletdomain "github.com/dujiao-next/internal/modules/wallet/domain"
)

// Repository owns all wallet aggregate persistence. Transactional callers
// receive the same port bound to their transaction through Transaction.
type Repository interface {
	GetAccountByUserID(userID uint) (*walletdomain.Account, error)
	GetAccountByUserIDForUpdate(userID uint) (*walletdomain.Account, error)
	GetAccountsByUserIDs(userIDs []uint) ([]walletdomain.Account, error)
	CreateAccount(account *walletdomain.Account) error
	UpdateAccount(account *walletdomain.Account) error
	ListAccounts(filter AccountListFilter) ([]walletdomain.Account, int64, error)

	CreateTransaction(transaction *walletdomain.Transaction) error
	GetTransactionByReference(reference string) (*walletdomain.Transaction, error)
	CountOrderTransactionsByType(orderID uint, transactionType string) (int64, error)
	ListTransactions(filter TransactionListFilter) ([]walletdomain.Transaction, int64, error)

	ListRechargeRefunds(id uint) ([]walletdomain.Transaction, error)
	GetRechargeOrderByID(id uint) (*walletdomain.RechargeOrder, error)
	CreateRechargeOrder(order *walletdomain.RechargeOrder) error
	UpdateRechargeOrder(order *walletdomain.RechargeOrder) error
	ApplyRechargeRefund(userID uint, amount decimal.Decimal) error
	GetRechargeOrderByRechargeNo(userID uint, rechargeNo string) (*walletdomain.RechargeOrder, error)
	GetRechargeOrderByPaymentID(paymentID uint) (*walletdomain.RechargeOrder, error)
	GetRechargeOrderByPaymentIDAndUser(paymentID, userID uint) (*walletdomain.RechargeOrder, error)
	GetRechargeOrderByPaymentIDForUpdate(paymentID uint) (*walletdomain.RechargeOrder, error)
	GetRechargeOrdersByPaymentIDs(paymentIDs []uint) ([]walletdomain.RechargeOrder, error)
	ListRechargeOrdersAdmin(filter RechargeListFilter) ([]walletdomain.RechargeOrder, int64, error)
	StatsRechargeOrders(filter RechargeListFilter) (map[string]int64, error)

	CreateUnifiedOp(op *walletdomain.UnifiedOp) error
	GetUnifiedOpByOpID(opID string) (*walletdomain.UnifiedOp, error)
	GetActiveOrderDebitUnifiedOpForUpdate(orderID uint) (*walletdomain.UnifiedOp, error)
	UpdateUnifiedOp(op *walletdomain.UnifiedOp) error
	ListPendingUnifiedOps(now time.Time, orphanBefore time.Time, limit int) ([]walletdomain.UnifiedOp, error)
	ListUnresolvedUnifiedOps(limit int) ([]walletdomain.UnifiedOp, error)
}

// Transaction is the wallet-owned view of an already-open database
// transaction. It intentionally exposes no ORM primitive.
type Transaction interface {
	Wallets() Repository
}

type UnitOfWork interface {
	WithinTransaction(fn func(Transaction) error) error
}

// RechargeRefundEmailQueue 将已完成的充值退款邮件加入异步队列。
type RechargeRefundEmailQueue interface {
	EnqueueRechargeRefundEmail(rechargeID, refundID uint, status string) error
}

type UseCase interface {
	GetAccount(userID uint) (*walletdomain.Account, error)
	ListTransactions(filter TransactionListFilter) ([]walletdomain.Transaction, int64, error)
	ListRechargeOrdersAdmin(filter RechargeListFilter) ([]walletdomain.RechargeOrder, int64, error)
	ListUserRechargeOrders(userID uint, page, pageSize int, status, rechargeNo string) ([]walletdomain.RechargeOrder, int64, error)
	StatsUserRechargeOrders(userID uint, rechargeNo string) (map[string]int64, error)
	GetRechargeOrderByRechargeNo(userID uint, rechargeNo string) (*walletdomain.RechargeOrder, error)
	GetRechargeOrderByPaymentIDAndUser(paymentID, userID uint) (*walletdomain.RechargeOrder, error)
	GetBalancesByUserIDs(userIDs []uint) (map[uint]money.Amount, error)

	Recharge(input RechargeInput) (*walletdomain.Account, *walletdomain.Transaction, error)
	AdminAdjustBalance(input AdjustBalanceInput) (*walletdomain.Account, *walletdomain.Transaction, error)
	CreditInTransaction(tx Transaction, input CreditInput) (*walletdomain.Account, *walletdomain.Transaction, error)
	ApplyRechargePayment(tx Transaction, recharge *walletdomain.RechargeOrder) (*walletdomain.Transaction, error)
	ApplyOrderBalance(tx Transaction, input OrderBalanceInput) (money.Amount, error)
	ReleaseOrderBalance(tx Transaction, input OrderReleaseInput, claim ReleaseClaim) (money.Amount, error)
}

// ReleaseClaim atomically clears the order-side wallet allocation before the
// wallet credit is written. Returning false means another attempt already won.
type ReleaseClaim func(now time.Time) (bool, error)

// UnifiedBalanceRedeemResult holds the result of redeeming a gift card or
// redemption code directly into the unified NeedToken API balance.
type UnifiedBalanceRedeemResult struct {
	Source       string
	Credited     money.Amount
	BalanceAfter money.Amount
	Currency     string
}

// UnifiedBalanceBridge synchronizes storefront wallet reads, credits, debits,
// and redemptions with the authoritative NeedToken API User.Quota ledger.
// All bridge calls MUST be invoked outside local SQLite/GORM transactions.
type UnifiedBalanceBridge interface {
	Enabled() bool
	SiteCurrency() string
	FetchBalance(ctx context.Context, userID uint, siteCurrency string) (money.Amount, bool, error)
	Credit(ctx context.Context, userID uint, amount money.Amount, currency, operationID, remark string) (money.Amount, error)
	Debit(ctx context.Context, userID uint, maxAmount money.Amount, currency string, allowPartial bool, operationID, remark string) (debited money.Amount, balanceAfter money.Amount, replayed bool, err error)
	Redeem(ctx context.Context, userID uint, code, siteCurrency string) (*UnifiedBalanceRedeemResult, error)
}

// UnifiedOpAuthorityStatus describes the authoritative NeedToken API state of
// a cross-system wallet operation queried by operation_id.
type UnifiedOpAuthorityStatus struct {
	Found   bool
	Status  string // "not_found", "debited", "rolled_back", "credited"
	Amount  money.Amount
	Balance money.Amount
}

// UnifiedBalanceOpQuerier is an optional extension of UnifiedBalanceBridge that
// queries authoritative operation state from NeedToken API before recovering
// uncertain pending operations.
type UnifiedBalanceOpQuerier interface {
	QueryOp(ctx context.Context, userID uint, operationID, kind, currency string) (*UnifiedOpAuthorityStatus, error)
}

// UnifiedDebitOrderReconciler reconciles a confirmed NewAPI debit against the
// target storefront order during background recovery.
type UnifiedDebitOrderReconciler interface {
	ReconcileUnifiedDebitOrder(op *walletdomain.UnifiedOp) (applied bool, shouldRollback bool, err error)
}
