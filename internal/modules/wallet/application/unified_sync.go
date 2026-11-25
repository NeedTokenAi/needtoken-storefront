package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dujiao-next/internal/logger"
	walletcontract "github.com/dujiao-next/internal/modules/wallet/contract"
	walletdomain "github.com/dujiao-next/internal/modules/wallet/domain"
	"github.com/dujiao-next/internal/shared/money"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	unifiedOpInFlightLease   = 30 * time.Second
	unifiedOpOrphanDebitAge  = 5 * time.Minute
	unifiedOpMaxRetryBackoff = 15 * time.Minute
)

// EnqueueUnifiedCreditInTx records a durable credit outbox item inside the
// caller's open DB transaction. Callers MUST invoke FlushUnifiedCreditOp after
// the transaction commits so HTTP never runs under SQLite/DB locks.
func (s *Service) EnqueueUnifiedCreditInTx(
	tx walletcontract.Transaction,
	userID uint,
	orderID *uint,
	amount money.Amount,
	currency, opID, remark string,
) (*walletdomain.UnifiedOp, error) {
	if !s.HasUnifiedBalance() || userID == 0 || amount.Decimal.Round(2).LessThanOrEqual(decimal.Zero) {
		return nil, nil
	}
	if tx == nil || tx.Wallets() == nil {
		return nil, walletcontract.ErrTransactionRequired
	}
	opID = strings.TrimSpace(opID)
	if opID == "" {
		return nil, walletcontract.ErrTransactionCreateFailed
	}
	repo := tx.Wallets()
	existing, err := repo.GetUnifiedOpByOpID(opID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	now := time.Now()
	op := &walletdomain.UnifiedOp{
		OpID:      opID,
		Direction: walletdomain.UnifiedOpDirectionCredit,
		Status:    walletdomain.UnifiedOpStatusCreditPending,
		UserID:    userID,
		OrderID:   orderID,
		Amount:    money.FromDecimal(amount.Decimal.Round(2)),
		Currency:  normalizeCurrency(currency),
		Remark:    cleanRemark(remark, "统一余额入账"),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.CreateUnifiedOp(op); err != nil {
		return nil, err
	}
	return op, nil
}

// FlushUnifiedCreditOp delivers a persisted credit outbox op to NewAPI outside
// any local DB transaction. Failures leave the op in credit_pending with backoff
// for background recovery.
func (s *Service) FlushUnifiedCreditOp(op *walletdomain.UnifiedOp) {
	if s == nil || op == nil || !s.HasUnifiedBalance() {
		return
	}
	if op.Status == walletdomain.UnifiedOpStatusCredited || op.Status == walletdomain.UnifiedOpStatusRolledBack {
		return
	}
	_ = s.deliverUnifiedCreditOrRollback(op)
}

// ReserveOrderUnifiedDebitInTx acquires or resumes a durable per-order debit
// operation inside the caller's locked order transaction.
//
// Returns (op, alreadyDebited, err). Concurrent in-flight attempts fail closed
// with ErrUnifiedOpInProgress so two requests can never share an op_id or roll
// back each other's payment.
func (s *Service) ReserveOrderUnifiedDebitInTx(
	tx walletcontract.Transaction,
	userID, orderID uint,
	maxAmount money.Amount,
	currency, remark string,
) (*walletdomain.UnifiedOp, bool, error) {
	if !s.HasUnifiedBalance() || userID == 0 || orderID == 0 || maxAmount.Decimal.Round(2).LessThanOrEqual(decimal.Zero) {
		return nil, false, nil
	}
	if tx == nil || tx.Wallets() == nil {
		return nil, false, walletcontract.ErrTransactionRequired
	}
	repo := tx.Wallets()
	now := time.Now()
	existing, err := repo.GetActiveOrderDebitUnifiedOpForUpdate(orderID)
	if err != nil {
		return nil, false, err
	}
	if existing != nil {
		switch existing.Status {
		case walletdomain.UnifiedOpStatusDebited:
			existing.UpdatedAt = now
			if err := repo.UpdateUnifiedOp(existing); err != nil {
				return nil, false, err
			}
			return existing, true, nil
		case walletdomain.UnifiedOpStatusRollbackPending:
			return nil, false, walletcontract.ErrUnifiedOpInProgress
		case walletdomain.UnifiedOpStatusPending:
			// Still in flight unless the previous attempt explicitly settled with
			// an error (NextRetryAt != nil) or its lease expired after a crash.
			if existing.NextRetryAt == nil && now.Sub(existing.UpdatedAt) < unifiedOpInFlightLease {
				return nil, false, walletcontract.ErrUnifiedOpInProgress
			}
			// Reuse the exact unfinished op_id so an uncertain timeout resolves
			// idempotently without double-debiting NewAPI.
			existing.UpdatedAt = now
			existing.NextRetryAt = nil
			existing.LastError = ""
			if err := repo.UpdateUnifiedOp(existing); err != nil {
				return nil, false, err
			}
			return existing, false, nil
		}
	}

	oid := orderID
	op := &walletdomain.UnifiedOp{
		OpID:      fmt.Sprintf("order:%d:debit:%s", orderID, strings.ReplaceAll(uuid.NewString(), "-", "")),
		Direction: walletdomain.UnifiedOpDirectionDebit,
		Status:    walletdomain.UnifiedOpStatusPending,
		UserID:    userID,
		OrderID:   &oid,
		Amount:    money.FromDecimal(maxAmount.Decimal.Round(2)),
		Currency:  normalizeCurrency(currency),
		Remark:    cleanRemark(remark, "订单余额支付"),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.CreateUnifiedOp(op); err != nil {
		return nil, false, err
	}
	return op, false, nil
}

// ExecuteUnifiedDebit performs the external NewAPI debit call for a reserved op
// outside any DB transaction, then durably transitions the op to debited or applied.
func (s *Service) ExecuteUnifiedDebit(op *walletdomain.UnifiedOp, allowPartial bool) (money.Amount, money.Amount, error) {
	zero := money.FromDecimal(decimal.Zero)
	if s == nil || op == nil || !s.HasUnifiedBalance() {
		return zero, zero, nil
	}
	if op.Status == walletdomain.UnifiedOpStatusDebited {
		return op.Amount, zero, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	debited, balanceAfter, _, err := s.unified.Debit(ctx, op.UserID, op.Amount, op.Currency, allowPartial, op.OpID, op.Remark)
	if err != nil {
		s.markUnifiedDebitAttemptFailed(op, err)
		return zero, zero, err
	}

	now := time.Now()
	debitedRounded := money.FromDecimal(debited.Decimal.Round(2))
	if debitedRounded.Decimal.LessThanOrEqual(decimal.Zero) {
		op.Amount = zero
		op.Status = walletdomain.UnifiedOpStatusApplied
		op.LastError = ""
		op.NextRetryAt = nil
		op.UpdatedAt = now
		if updateErr := s.saveUnifiedOp(op); updateErr != nil {
			logger.Warnw("unified_wallet_mark_zero_debit_failed", "op_id", op.OpID, "error", updateErr)
		}
		s.updateLocalMirrorBalance(op.UserID, balanceAfter)
		return zero, balanceAfter, nil
	}

	op.Amount = debitedRounded
	op.Status = walletdomain.UnifiedOpStatusDebited
	op.LastError = ""
	op.NextRetryAt = nil
	op.UpdatedAt = now
	if updateErr := s.saveUnifiedOp(op); updateErr != nil {
		// NewAPI already debited; compensate immediately so funds are not stranded.
		s.CompensateUnifiedDebit(op, "本地记录统一余额扣款状态失败，退回余额")
		return zero, zero, updateErr
	}
	preDebitMirror := money.FromDecimal(balanceAfter.Decimal.Add(debitedRounded.Decimal).Round(2))
	s.updateLocalMirrorBalance(op.UserID, preDebitMirror)
	return debitedRounded, balanceAfter, nil
}

// MarkUnifiedDebitAppliedInTx marks a debited op as applied inside the same DB
// transaction that commits the order or wallet change.
func (s *Service) MarkUnifiedDebitAppliedInTx(tx walletcontract.Transaction, opID string) error {
	if !s.HasUnifiedBalance() || strings.TrimSpace(opID) == "" {
		return nil
	}
	if tx == nil || tx.Wallets() == nil {
		return walletcontract.ErrTransactionRequired
	}
	op, err := tx.Wallets().GetUnifiedOpByOpID(opID)
	if err != nil || op == nil {
		return err
	}
	if op.Status == walletdomain.UnifiedOpStatusApplied {
		return nil
	}
	op.Status = walletdomain.UnifiedOpStatusApplied
	op.LastError = ""
	op.NextRetryAt = nil
	op.UpdatedAt = time.Now()
	return tx.Wallets().UpdateUnifiedOp(op)
}

// CompensateUnifiedDebit durably transitions a debited op to rollback_pending
// and immediately attempts to credit NewAPI with op_id+":rollback". Any failure
// stays persisted for RecoverUnifiedOps to retry idempotently.
func (s *Service) CompensateUnifiedDebit(op *walletdomain.UnifiedOp, reason string) {
	if s == nil || op == nil || !s.HasUnifiedBalance() {
		return
	}
	if op.Amount.Decimal.Round(2).LessThanOrEqual(decimal.Zero) {
		return
	}
	if reason = strings.TrimSpace(reason); reason != "" {
		op.Remark = reason
	}
	now := time.Now()
	if s.transactions != nil {
		_ = s.transactions.WithinTransaction(func(tx walletcontract.Transaction) error {
			current, err := tx.Wallets().GetUnifiedOpByOpID(op.OpID)
			if err != nil || current == nil {
				return err
			}
			if current.Status == walletdomain.UnifiedOpStatusApplied || current.Status == walletdomain.UnifiedOpStatusRolledBack {
				op.Status = current.Status
				return nil
			}
			current.Status = walletdomain.UnifiedOpStatusRollbackPending
			if reason != "" {
				current.Remark = reason
			}
			current.NextRetryAt = nil
			current.UpdatedAt = now
			if err := tx.Wallets().UpdateUnifiedOp(current); err != nil {
				return err
			}
			*op = *current
			return nil
		})
	} else {
		op.Status = walletdomain.UnifiedOpStatusRollbackPending
		op.UpdatedAt = now
	}
	if op.Status != walletdomain.UnifiedOpStatusRollbackPending {
		return
	}
	_ = s.deliverUnifiedCreditOrRollback(op)
}

// ReserveAndDebitStrictForAdminOp executes a strict (allow_partial=false)
// authoritative NewAPI debit backed by a durable UnifiedOp for admin balance
// deductions and recharge refunds.
func (s *Service) ReserveAndDebitStrictForAdminOp(userID uint, amount money.Amount, currency, opID, remark string) (*walletdomain.UnifiedOp, error) {
	if !s.HasUnifiedBalance() || userID == 0 || amount.Decimal.Round(2).LessThanOrEqual(decimal.Zero) {
		return nil, nil
	}
	opID = strings.TrimSpace(opID)
	if opID == "" {
		return nil, walletcontract.ErrTransactionCreateFailed
	}
	now := time.Now()
	var op *walletdomain.UnifiedOp
	if err := s.transactions.WithinTransaction(func(tx walletcontract.Transaction) error {
		repo := tx.Wallets()
		existing, err := repo.GetUnifiedOpByOpID(opID)
		if err != nil {
			return err
		}
		if existing != nil {
			if existing.Status == walletdomain.UnifiedOpStatusApplied {
				op = existing
				return nil
			}
			if existing.Status == walletdomain.UnifiedOpStatusRollbackPending {
				return walletcontract.ErrUnifiedOpInProgress
			}
			if existing.Status == walletdomain.UnifiedOpStatusPending && existing.NextRetryAt == nil && now.Sub(existing.UpdatedAt) < unifiedOpInFlightLease {
				return walletcontract.ErrUnifiedOpInProgress
			}
			existing.UpdatedAt = now
			existing.NextRetryAt = nil
			if err := repo.UpdateUnifiedOp(existing); err != nil {
				return err
			}
			op = existing
			return nil
		}
		created := &walletdomain.UnifiedOp{
			OpID:      opID,
			Direction: walletdomain.UnifiedOpDirectionDebit,
			Status:    walletdomain.UnifiedOpStatusPending,
			UserID:    userID,
			Amount:    money.FromDecimal(amount.Decimal.Round(2)),
			Currency:  normalizeCurrency(currency),
			Remark:    cleanRemark(remark, "管理员扣减余额"),
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := repo.CreateUnifiedOp(created); err != nil {
			return err
		}
		op = created
		return nil
	}); err != nil {
		return nil, err
	}
	if op == nil || op.Status == walletdomain.UnifiedOpStatusApplied || op.Status == walletdomain.UnifiedOpStatusDebited {
		return op, nil
	}
	debited, _, err := s.ExecuteUnifiedDebit(op, false)
	if err != nil {
		return nil, err
	}
	if debited.Decimal.Round(2).LessThan(amount.Decimal.Round(2)) {
		s.CompensateUnifiedDebit(op, "统一余额扣减不足，自动退回")
		return nil, walletcontract.ErrInsufficientBalance
	}
	return op, nil
}

// RecoverUnifiedOps retries pending unified credits/rollbacks and compensates
// orphaned debited ops left behind by process crashes.
func (s *Service) RecoverUnifiedOps() error {
	if s == nil || !s.HasUnifiedBalance() || s.repository == nil {
		return nil
	}
	now := time.Now()
	rows, err := s.repository.ListPendingUnifiedOps(now, now.Add(-unifiedOpOrphanDebitAge), 50)
	if err != nil {
		return err
	}
	for i := range rows {
		op := rows[i]
		if op.Status == walletdomain.UnifiedOpStatusDebited {
			// Process crashed after NewAPI debit and before committing order/wallet tx.
			s.CompensateUnifiedDebit(&op, "异常中断扣款自动退回")
			continue
		}
		_ = s.deliverUnifiedCreditOrRollback(&op)
	}
	return nil
}

func (s *Service) deliverUnifiedCreditOrRollback(op *walletdomain.UnifiedOp) error {
	if s == nil || op == nil || !s.HasUnifiedBalance() {
		return nil
	}
	targetOpID := op.OpID
	targetStatus := walletdomain.UnifiedOpStatusCredited
	if op.Status == walletdomain.UnifiedOpStatusRollbackPending {
		targetOpID = op.OpID + ":rollback"
		targetStatus = walletdomain.UnifiedOpStatusRolledBack
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	balanceAfter, err := s.unified.Credit(ctx, op.UserID, op.Amount, op.Currency, targetOpID, op.Remark)
	now := time.Now()
	if err != nil {
		op.Attempts++
		op.LastError = truncateUnifiedOpError(err.Error())
		next := now.Add(unifiedOpRetryBackoff(op.Attempts))
		op.NextRetryAt = &next
		op.UpdatedAt = now
		_ = s.saveUnifiedOp(op)
		logger.Warnw("unified_wallet_outbox_delivery_failed",
			"op_id", targetOpID,
			"status", op.Status,
			"attempts", op.Attempts,
			"error", err,
		)
		return err
	}
	op.Status = targetStatus
	op.LastError = ""
	op.NextRetryAt = nil
	op.UpdatedAt = now
	if saveErr := s.saveUnifiedOp(op); saveErr != nil {
		logger.Warnw("unified_wallet_outbox_mark_done_failed", "op_id", targetOpID, "error", saveErr)
	}
	s.updateLocalMirrorBalance(op.UserID, balanceAfter)
	return nil
}

func (s *Service) markUnifiedDebitAttemptFailed(op *walletdomain.UnifiedOp, debitErr error) {
	if op == nil {
		return
	}
	now := time.Now()
	op.Attempts++
	op.LastError = truncateUnifiedOpError(debitErr.Error())
	op.UpdatedAt = now
	// Mark retryable immediately so a user retry reuses the same op_id instead of
	// being blocked by the in-flight lease, while keeping the durable record.
	op.NextRetryAt = &now
	_ = s.saveUnifiedOp(op)
}

func (s *Service) saveUnifiedOp(op *walletdomain.UnifiedOp) error {
	if s == nil || op == nil || s.transactions == nil {
		return nil
	}
	return s.transactions.WithinTransaction(func(tx walletcontract.Transaction) error {
		return tx.Wallets().UpdateUnifiedOp(op)
	})
}

func unifiedOpRetryBackoff(attempts int) time.Duration {
	if attempts <= 1 {
		return 15 * time.Second
	}
	backoff := 15 * time.Second
	for i := 1; i < attempts && backoff < unifiedOpMaxRetryBackoff; i++ {
		backoff *= 2
	}
	if backoff > unifiedOpMaxRetryBackoff {
		return unifiedOpMaxRetryBackoff
	}
	return backoff
}

func truncateUnifiedOpError(msg string) string {
	msg = strings.TrimSpace(msg)
	if len(msg) <= 480 {
		return msg
	}
	return msg[:480]
}
