package gormstore

import (
	"errors"
	"strings"
	"time"

	walletdomain "github.com/dujiao-next/internal/modules/wallet/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Store) CreateUnifiedOp(op *walletdomain.UnifiedOp) error {
	return s.db.Create(op).Error
}

func (s *Store) GetUnifiedOpByOpID(opID string) (*walletdomain.UnifiedOp, error) {
	opID = strings.TrimSpace(opID)
	if opID == "" {
		return nil, nil
	}
	var op walletdomain.UnifiedOp
	if err := s.db.Where("op_id = ?", opID).First(&op).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &op, nil
}

func (s *Store) GetActiveOrderDebitUnifiedOpForUpdate(orderID uint) (*walletdomain.UnifiedOp, error) {
	if orderID == 0 {
		return nil, nil
	}
	var op walletdomain.UnifiedOp
	if err := s.db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("order_id = ? AND direction = ? AND status IN ?", orderID, walletdomain.UnifiedOpDirectionDebit, []string{
			walletdomain.UnifiedOpStatusPending,
			walletdomain.UnifiedOpStatusDebited,
			walletdomain.UnifiedOpStatusRollbackPending,
		}).
		Order("id DESC").
		First(&op).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &op, nil
}

func (s *Store) UpdateUnifiedOp(op *walletdomain.UnifiedOp) error {
	return s.db.Save(op).Error
}

func (s *Store) ListPendingUnifiedOps(now time.Time, orphanBefore time.Time, limit int) ([]walletdomain.UnifiedOp, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []walletdomain.UnifiedOp
	err := s.db.Where(
		"((status IN ? AND (next_retry_at IS NULL OR next_retry_at <= ?)) OR (status = ? AND updated_at <= ?))",
		[]string{walletdomain.UnifiedOpStatusCreditPending, walletdomain.UnifiedOpStatusRollbackPending},
		now,
		walletdomain.UnifiedOpStatusDebited,
		orphanBefore,
	).Order("id ASC").Limit(limit).Find(&rows).Error
	return rows, err
}
