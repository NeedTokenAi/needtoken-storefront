package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dujiao-next/internal/logger"
	settingscontract "github.com/dujiao-next/internal/modules/settings/contract"
	walletcontract "github.com/dujiao-next/internal/modules/wallet/contract"
	walletdomain "github.com/dujiao-next/internal/modules/wallet/domain"
	"github.com/dujiao-next/internal/shared/money"
	"github.com/shopspring/decimal"
)

const defaultCurrency = "CNY"

type Options struct {
	Repository   walletcontract.Repository
	Transactions walletcontract.UnitOfWork
	RefundConfig settingscontract.OrderRefundConfigReader
	RefundEmails walletcontract.RechargeRefundEmailQueue
	Unified      walletcontract.UnifiedBalanceBridge
}

type Service struct {
	repository   walletcontract.Repository
	transactions walletcontract.UnitOfWork
	refundConfig settingscontract.OrderRefundConfigReader
	refundEmails walletcontract.RechargeRefundEmailQueue
	unified      walletcontract.UnifiedBalanceBridge
}

var _ walletcontract.UseCase = (*Service)(nil)

func NewService(options Options) *Service {
	return &Service{
		repository:   options.Repository,
		transactions: options.Transactions,
		refundConfig: options.RefundConfig,
		refundEmails: options.RefundEmails,
		unified:      options.Unified,
	}
}

// SetUnifiedBalanceBridge wires the optional unified NeedToken API balance ledger.
func (s *Service) SetUnifiedBalanceBridge(bridge walletcontract.UnifiedBalanceBridge) {
	if s == nil {
		return
	}
	s.unified = bridge
}

// HasUnifiedBalance reports whether the authoritative NeedToken API balance bridge is active.
func (s *Service) HasUnifiedBalance() bool {
	return s != nil && s.unified != nil && s.unified.Enabled()
}

// SyncLocalBalanceFromUnified fetches the live NeedToken API User.Quota balance
// outside any DB transaction and mirrors it into the local wallet_accounts row.
func (s *Service) SyncLocalBalanceFromUnified(userID uint) (*walletdomain.Account, error) {
	if !s.HasUnifiedBalance() || userID == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	amount, ok, err := s.unified.FetchBalance(ctx, userID, s.unified.SiteCurrency())
	if err != nil || !ok {
		return nil, err
	}
	var updated *walletdomain.Account
	if s.transactions != nil {
		_ = s.transactions.WithinTransaction(func(tx walletcontract.Transaction) error {
			now := time.Now()
			acc, accErr := ensureAccountForUpdate(tx.Wallets(), userID, now)
			if accErr != nil {
				return accErr
			}
			acc.Balance = money.FromDecimal(amount.Decimal.Round(2))
			acc.UpdatedAt = now
			if err := tx.Wallets().UpdateAccount(acc); err != nil {
				return err
			}
			updated = acc
			return nil
		})
	}
	if updated == nil {
		updated = &walletdomain.Account{UserID: userID, Balance: money.FromDecimal(amount.Decimal.Round(2)), UpdatedAt: time.Now()}
	}
	return updated, nil
}

// SyncCreditToUnified idempotently credits the authoritative NeedToken API User.Quota
// outside any local DB transaction.
func (s *Service) SyncCreditToUnified(userID uint, amount money.Amount, currency, operationID, remark string) {
	if !s.HasUnifiedBalance() || userID == 0 || amount.Decimal.Round(2).LessThanOrEqual(decimal.Zero) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	balanceAfter, err := s.unified.Credit(ctx, userID, amount, currency, operationID, remark)
	if err != nil {
		logger.Warnw("unified_wallet_credit_sync_failed", "user_id", userID, "operation_id", operationID, "amount", amount.String(), "error", err)
		return
	}
	s.updateLocalMirrorBalance(userID, balanceAfter)
}

// SyncDebitFromUnified idempotently debits the authoritative NeedToken API User.Quota
// outside any local DB transaction and mirrors the pre-debit balance locally so the
// subsequent short order transaction allocates the exact debited amount.
func (s *Service) SyncDebitFromUnified(userID uint, maxAmount money.Amount, currency string, allowPartial bool, operationID, remark string) (money.Amount, error) {
	if !s.HasUnifiedBalance() || userID == 0 || maxAmount.Decimal.Round(2).LessThanOrEqual(decimal.Zero) {
		return money.FromDecimal(decimal.Zero), nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	debited, balanceAfter, err := s.unified.Debit(ctx, userID, maxAmount, currency, allowPartial, operationID, remark)
	if err != nil {
		return money.FromDecimal(decimal.Zero), err
	}
	// Mirror balance BEFORE the local ApplyOrderBalance debit so local account
	// ends at balanceAfter once ApplyOrderBalance subtracts debited.
	preDebitMirror := money.FromDecimal(balanceAfter.Decimal.Add(debited.Decimal).Round(2))
	s.updateLocalMirrorBalance(userID, preDebitMirror)
	return debited, nil
}

func (s *Service) updateLocalMirrorBalance(userID uint, balance money.Amount) {
	if s == nil || s.transactions == nil || userID == 0 {
		return
	}
	_ = s.transactions.WithinTransaction(func(tx walletcontract.Transaction) error {
		now := time.Now()
		acc, err := ensureAccountForUpdate(tx.Wallets(), userID, now)
		if err != nil {
			return err
		}
		acc.Balance = money.FromDecimal(balance.Decimal.Round(2))
		acc.UpdatedAt = now
		return tx.Wallets().UpdateAccount(acc)
	})
}

// getRechargeRefundConfig 在进入钱包事务前读取退款期限配置。
func (s *Service) getRechargeRefundConfig() (settingscontract.OrderRefundConfig, error) {
	if s.refundConfig == nil {
		return settingscontract.DefaultOrderRefundConfig(), nil
	}
	return s.refundConfig.GetOrderRefundConfig()
}

func normalizeCurrency(currency string) string {
	normalized := strings.ToUpper(strings.TrimSpace(currency))
	if normalized == "" {
		return defaultCurrency
	}
	return normalized
}

func cleanRemark(raw, fallback string) string {
	remark := strings.TrimSpace(raw)
	if remark == "" {
		return fallback
	}
	return remark
}

// orderAllocationReference 生成订单余额分配的轮次幂等键。
//
// 第 1 轮沿用历史格式（order:<id>:<action>），升级前已经扣款的在途订单不会被重复扣款；
// 订单在"用余额 → 改在线（退回）→ 再用余额"之间来回切换时，后续轮次追加序号，
// 否则上一轮早已被退回的流水会把新一轮的扣款判成重复操作而跳过——
// 订单会被标记为已用余额，钱包却一分钱都没扣。
//
// 同一轮内的重复调用由订单侧的 wallet_paid_amount 与 Release 的原子占位拦截，
// 并发写入还有 wallet_transactions.reference 唯一索引兜底。
func orderAllocationReference(repository walletcontract.Repository, orderID uint, action string) (string, error) {
	base := orderReference(orderID, action)
	if repository == nil || orderID == 0 {
		return base, nil
	}
	count, err := repository.CountOrderTransactionsByType(orderID, action)
	if err != nil {
		return "", err
	}
	if count <= 0 {
		return base, nil
	}
	return fmt.Sprintf("%s:%d", base, count+1), nil
}

func orderReference(orderID uint, action string) string {
	action = strings.TrimSpace(action)
	if action == "" {
		action = "wallet"
	}
	return fmt.Sprintf("order:%d:%s", orderID, action)
}

func uniqueReference(prefix string, id uint) string {
	normalized := strings.TrimSpace(prefix)
	if normalized == "" {
		normalized = "wallet"
	}
	return fmt.Sprintf("%s:%d:%d", normalized, id, time.Now().UnixNano())
}
