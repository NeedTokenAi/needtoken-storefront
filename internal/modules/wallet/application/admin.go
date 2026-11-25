package application

import (
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	walletcontract "github.com/dujiao-next/internal/modules/wallet/contract"
	walletdomain "github.com/dujiao-next/internal/modules/wallet/domain"
	"github.com/dujiao-next/internal/shared/money"

	"github.com/shopspring/decimal"
)

func (s *Service) Recharge(input walletcontract.RechargeInput) (*walletdomain.Account, *walletdomain.Transaction, error) {
	if input.UserID == 0 {
		return nil, nil, walletcontract.ErrAccountNotFound
	}
	amount := input.Amount.Decimal.Round(2)
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, nil, walletcontract.ErrInvalidAmount
	}
	ref := uniqueReference("recharge", input.UserID)
	remark := cleanRemark(input.Remark, "用户充值")
	currency := normalizeCurrency(input.Currency)
	acc, txn, creditOp, err := s.changeBalanceWithUnified(
		input.UserID, amount, constants.WalletTxnTypeRecharge, nil,
		ref, remark, currency, nil, "",
	)
	if err != nil {
		return nil, nil, err
	}
	s.FlushUnifiedCreditOp(creditOp)
	return acc, txn, nil
}

func (s *Service) AdminAdjustBalance(input walletcontract.AdjustBalanceInput) (*walletdomain.Account, *walletdomain.Transaction, error) {
	if input.UserID == 0 || input.OperatorAdminID == 0 {
		return nil, nil, walletcontract.ErrAccountNotFound
	}
	delta := input.Delta.Decimal.Round(2)
	if delta.IsZero() {
		return nil, nil, walletcontract.ErrInvalidAmount
	}
	ref := uniqueReference("admin_adjust", input.UserID)
	remark := cleanRemark(input.Remark, "管理员调整余额")
	currency := normalizeCurrency(input.Currency)

	if delta.LessThan(decimal.Zero) && s.HasUnifiedBalance() {
		absAmount := money.FromDecimal(delta.Abs().Round(2))
		debitOp, err := s.ReserveAndDebitStrictForAdminOp(input.UserID, absAmount, currency, ref, remark)
		if err != nil {
			return nil, nil, err
		}
		acc, txn, _, txErr := s.changeBalanceWithUnified(
			input.UserID, delta, constants.WalletTxnTypeAdminAdjust, nil,
			ref, remark, currency, &input.OperatorAdminID, debitOp.OpID,
		)
		if txErr != nil {
			s.CompensateUnifiedDebit(debitOp, "管理员调额本地事务失败，退回余额")
			return nil, nil, txErr
		}
		return acc, txn, nil
	}

	acc, txn, creditOp, err := s.changeBalanceWithUnified(
		input.UserID, delta, constants.WalletTxnTypeAdminAdjust, nil,
		ref, remark, currency, &input.OperatorAdminID, "",
	)
	if err != nil {
		return nil, nil, err
	}
	s.FlushUnifiedCreditOp(creditOp)
	return acc, txn, nil
}

func (s *Service) changeBalanceWithUnified(
	userID uint,
	delta decimal.Decimal,
	transactionType string,
	orderID *uint,
	reference, remark, currency string,
	operatorAdminID *uint,
	appliedDebitOpID string,
) (*walletdomain.Account, *walletdomain.Transaction, *walletdomain.UnifiedOp, error) {
	if s.transactions == nil {
		return nil, nil, nil, walletcontract.ErrTransactionRequired
	}
	var accountResult *walletdomain.Account
	var transactionResult *walletdomain.Transaction
	var creditOpResult *walletdomain.UnifiedOp
	if err := s.transactions.WithinTransaction(func(tx walletcontract.Transaction) error {
		repository := tx.Wallets()
		now := time.Now()
		account, err := ensureAccountForUpdate(repository, userID, now)
		if err != nil {
			return err
		}

		before := account.Balance.Decimal.Round(2)
		if appliedDebitOpID != "" && delta.LessThan(decimal.Zero) && before.LessThan(delta.Abs().Round(2)) {
			// NewAPI authoritative debit already succeeded; align local mirror before subtracting.
			before = delta.Abs().Round(2)
		}
		after := before.Add(delta).Round(2)
		if after.LessThan(decimal.Zero) {
			return walletcontract.ErrInsufficientBalance
		}
		direction := constants.WalletTxnDirectionIn
		amount := delta.Round(2)
		if delta.LessThan(decimal.Zero) {
			direction = constants.WalletTxnDirectionOut
			amount = delta.Abs().Round(2)
		}

		account.Balance = money.FromDecimal(after)
		account.UpdatedAt = now
		if err := repository.UpdateAccount(account); err != nil {
			return walletcontract.ErrAccountUpdateFailed
		}
		trimmedRef := strings.TrimSpace(reference)
		transaction := &walletdomain.Transaction{
			UserID: userID, OperatorAdminID: operatorAdminID, OrderID: orderID, Type: transactionType, Direction: direction,
			Amount: money.FromDecimal(amount), BalanceBefore: money.FromDecimal(before),
			BalanceAfter: money.FromDecimal(after), Currency: normalizeCurrency(currency),
			Reference: trimmedRef, Remark: remark, CreatedAt: now, UpdatedAt: now,
		}
		if err := repository.CreateTransaction(transaction); err != nil {
			return walletcontract.ErrTransactionCreateFailed
		}
		if appliedDebitOpID != "" {
			if err := s.MarkUnifiedDebitAppliedInTx(tx, appliedDebitOpID); err != nil {
				return err
			}
		} else if delta.GreaterThan(decimal.Zero) && s.HasUnifiedBalance() {
			op, err := s.EnqueueUnifiedCreditInTx(tx, userID, orderID, money.FromDecimal(amount), currency, trimmedRef, remark)
			if err != nil {
				return err
			}
			creditOpResult = op
		}
		accountResult, transactionResult = account, transaction
		return nil
	}); err != nil {
		return nil, nil, nil, err
	}
	return accountResult, transactionResult, creditOpResult, nil
}
