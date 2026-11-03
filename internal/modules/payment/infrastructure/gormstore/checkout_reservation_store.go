package gormstore

import (
	"encoding/json"
	"errors"
	"strconv"

	paymentcontract "github.com/dujiao-next/internal/modules/payment/contract"
	paymentdomain "github.com/dujiao-next/internal/modules/payment/domain"
	"github.com/dujiao-next/internal/shared/jsonmap"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Store) ListCheckoutReservations(rootOrderID uint) ([]paymentdomain.CheckoutReservation, error) {
	rows := []paymentdomain.CheckoutReservation{}
	err := s.db.Where("root_order_id = ?", rootOrderID).Order("order_item_id").Find(&rows).Error
	return rows, err
}

func (s *Store) CreateCheckoutReservation(record paymentdomain.CheckoutReservation) (*paymentdomain.CheckoutReservation, error) {
	if err := s.db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "order_item_id"}}, DoNothing: true}).Create(&record).Error; err != nil {
		return nil, err
	}
	var stored paymentdomain.CheckoutReservation
	if err := s.db.First(&stored, "order_item_id = ?", record.OrderItemID).Error; err != nil {
		return nil, err
	}
	if !sameReservationIdentity(stored, record) {
		return nil, paymentcontract.ErrReservationConflict
	}
	return &stored, nil
}

func sameReservationIdentity(a, b paymentdomain.CheckoutReservation) bool {
	aa, _ := json.Marshal(a.ReserveRequestJSON)
	bb, _ := json.Marshal(b.ReserveRequestJSON)
	return a.OrderItemID == b.OrderItemID && a.RootOrderID == b.RootOrderID && a.OrderID == b.OrderID && a.ConnectionID == b.ConnectionID && a.UpstreamSKUID == b.UpstreamSKUID && a.RequirementHash == b.RequirementHash && a.OrderRef == b.OrderRef && a.ReserveOperationID == b.ReserveOperationID && a.ReservePayloadHash == b.ReservePayloadHash && string(aa) == string(bb) && a.ExpiresAt.Equal(b.ExpiresAt)
}

func (s *Store) SaveCheckoutReservation(record paymentdomain.CheckoutReservation) (*paymentdomain.CheckoutReservation, error) {
	var result paymentdomain.CheckoutReservation
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var old paymentdomain.CheckoutReservation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&old, "order_item_id = ?", record.OrderItemID).Error; err != nil {
			return err
		}
		if !sameReservationIdentity(old, record) {
			return paymentcontract.ErrReservationConflict
		}
		if len(old.ConfirmRequestJSON) > 0 {
			a, _ := json.Marshal(old.ConfirmRequestJSON)
			b, _ := json.Marshal(record.ConfirmRequestJSON)
			if len(record.ConfirmRequestJSON) > 0 && string(a) != string(b) {
				return paymentcontract.ErrReservationConflict
			}
			record.ConfirmRequestJSON = old.ConfirmRequestJSON
		}
		if old.HoldID != "" && record.HoldID != "" && old.HoldID != record.HoldID {
			return paymentcontract.ErrReservationConflict
		}
		if old.UpstreamIdentity != "" {
			if record.UpstreamIdentity != "" && record.UpstreamIdentity != old.UpstreamIdentity {
				return paymentcontract.ErrReservationConflict
			}
			record.UpstreamIdentity = old.UpstreamIdentity
		}
		oldVersion, _ := strconv.ParseUint(old.HoldVersion, 10, 64)
		newVersion, _ := strconv.ParseUint(record.HoldVersion, 10, 64)
		// A delayed HTTP response cannot roll a newer paid/bound/expired
		// projection back to reserved, nor erase a known result with unknown.
		if newVersion < oldVersion || (old.HoldID != "" && record.HoldID == "") || (newVersion == oldVersion && old.State != "pending" && old.State != "unknown" && record.State == "unknown") || (newVersion == oldVersion && terminalReservation(old.State) && !terminalReservation(record.State)) {
			changed := false
			if newVersion == oldVersion && record.State == "unknown" {
				// Preserve the known hold identity/book version while recording
				// the latest failed observation. A cached ready=true must never
				// bypass the fail-closed in-transaction payment check.
				old.ReadyForPayment = false
				old.AttentionCode = record.AttentionCode
				old.UpdatedAt = record.UpdatedAt
				if old.State == "reserved" {
					old.State = "unknown"
				}
				changed = true
			}
			if len(old.ConfirmRequestJSON) == 0 && len(record.ConfirmRequestJSON) > 0 {
				old.ConfirmRequestJSON = record.ConfirmRequestJSON
				old.UpdatedAt = record.UpdatedAt
				changed = true
			}
			if changed {
				if err := tx.Save(&old).Error; err != nil {
					return err
				}
				if err := writeCheckoutReceipt(tx, old); err != nil {
					return err
				}
			}
			result = old
			return nil
		}
		if old.State == "needs_attention" && record.State == "reserved" {
			record.State = "needs_attention"
			record.ReadyForPayment = false
			record.AttentionCode = old.AttentionCode
		}
		if err := tx.Save(&record).Error; err != nil {
			return err
		}
		if err := writeCheckoutReceipt(tx, record); err != nil {
			return err
		}
		result = record
		return nil
	})
	return &result, err
}

// Save the harmless hold reference/version beside the frozen SKU promise in
// the same transaction as its index, including failed/unknown observations.
func writeCheckoutReceipt(tx *gorm.DB, record paymentdomain.CheckoutReservation) error {
	var item struct {
		SKUSnapshotJSON jsonmap.JSON `gorm:"column:sku_snapshot_json;type:json"`
	}
	q := tx.Table("order_items").Select("sku_snapshot_json").Where("id = ? AND order_id = ?", record.OrderItemID, record.OrderID).Take(&item)
	if errors.Is(q.Error, gorm.ErrRecordNotFound) {
		return paymentcontract.ErrReservationInvalid
	}
	if q.Error != nil {
		return q.Error
	}
	if item.SKUSnapshotJSON == nil {
		return paymentcontract.ErrReservationInvalid
	}
	item.SKUSnapshotJSON["reservation_hold"] = jsonmap.JSON{"order_ref": record.OrderRef, "hold_id": record.HoldID, "version": record.HoldVersion, "state": record.State, "expires_at": record.ExpiresAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00")}
	return tx.Table("order_items").Where("id = ? AND order_id = ?", record.OrderItemID, record.OrderID).Update("sku_snapshot_json", item.SKUSnapshotJSON).Error
}

func terminalReservation(state string) bool {
	return state == "paid" || state == "bound" || state == "consumed" || state == "expired" || state == "released" || state == "needs_attention"
}
