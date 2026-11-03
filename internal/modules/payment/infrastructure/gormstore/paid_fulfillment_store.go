package gormstore

import (
	"time"

	"github.com/dujiao-next/internal/constants"
	paymentdomain "github.com/dujiao-next/internal/modules/payment/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Store) RecordPaidFulfillment(orderID uint, now time.Time) error {
	event := paymentdomain.PaidFulfillmentEvent{OrderID: orderID, NextAttemptAt: now, CreatedAt: now, UpdatedAt: now}
	return s.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&event).Error
}

// Backfill only orders still awaiting fulfillment. Refunded/canceled historical
// orders must never trigger a new purchase during recovery.
func (s *Store) BackfillPaidFulfillment(now time.Time, limit int) error {
	var ids []uint
	err := s.db.Table("orders").Select("orders.id").
		Where("orders.deleted_at IS NULL AND orders.parent_id IS NULL AND orders.paid_at IS NOT NULL AND orders.status IN ?", []string{constants.OrderStatusPaid, constants.OrderStatusFulfilling, constants.OrderStatusPartiallyDelivered}).
		Where("NOT EXISTS (SELECT 1 FROM paid_fulfillment_events e WHERE e.order_id = orders.id)").
		Where("EXISTS (SELECT 1 FROM order_items i WHERE i.order_id = orders.id AND i.deleted_at IS NULL AND i.fulfillment_type = ?) OR EXISTS (SELECT 1 FROM orders c JOIN order_items i ON i.order_id = c.id WHERE c.parent_id = orders.id AND c.deleted_at IS NULL AND i.deleted_at IS NULL AND i.fulfillment_type = ?)", constants.FulfillmentTypeUpstream, constants.FulfillmentTypeUpstream).
		Order("orders.id").Limit(limit).Scan(&ids).Error
	if err != nil {
		return err
	}
	// The query is finished before writes; safe with SQLite MaxOpenConns=1.
	for _, id := range ids {
		if err := s.RecordPaidFulfillment(id, now); err != nil {
			return err
		}
	}
	return nil
}
func (s *Store) ListPendingPaidFulfillment(now time.Time, limit int) ([]paymentdomain.PaidFulfillmentEvent, error) {
	var events []paymentdomain.PaidFulfillmentEvent
	err := s.db.Where("completed_at IS NULL AND next_attempt_at <= ? AND (lease_until IS NULL OR lease_until <= ?)", now, now).Order("next_attempt_at, order_id").Limit(limit).Find(&events).Error
	return events, err
}
func (s *Store) ClaimPaidFulfillment(orderID uint, leaseID string, now time.Time) (bool, error) {
	until := now.Add(5 * time.Minute)
	result := s.db.Model(&paymentdomain.PaidFulfillmentEvent{}).Where("order_id = ? AND completed_at IS NULL AND next_attempt_at <= ? AND (lease_until IS NULL OR lease_until <= ?)", orderID, now, now).
		Updates(map[string]interface{}{"lease_id": leaseID, "lease_until": until, "attempts": gorm.Expr("attempts + 1"), "updated_at": now})
	return result.RowsAffected == 1, result.Error
}
func (s *Store) FinishPaidFulfillment(orderID uint, leaseID string, completed bool, now time.Time) error {
	updates := map[string]interface{}{"lease_id": "", "lease_until": nil, "next_attempt_at": now.Add(time.Minute), "updated_at": now}
	if completed {
		updates["completed_at"] = now
	}
	return s.db.Model(&paymentdomain.PaidFulfillmentEvent{}).Where("order_id = ? AND lease_id = ? AND completed_at IS NULL", orderID, leaseID).Updates(updates).Error
}
