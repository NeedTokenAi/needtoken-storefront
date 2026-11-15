package subscriptionauthorizationstore

import (
	"context"
	"errors"
	"time"

	ordercontract "github.com/dujiao-next/internal/modules/order/contract"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SubscriptionAuthorizationStore struct{ db *gorm.DB }

func NewSubscriptionAuthorizationStore(db *gorm.DB) *SubscriptionAuthorizationStore {
	return &SubscriptionAuthorizationStore{db: db}
}

func (s *SubscriptionAuthorizationStore) Get(ctx context.Context, id string) (*orderdomain.SubscriptionAuthorization, error) {
	var row orderdomain.SubscriptionAuthorization
	err := s.db.WithContext(ctx).First(&row, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ordercontract.ErrAuthorizationNotFound
	}
	return &row, err
}

func (s *SubscriptionAuthorizationStore) Create(ctx context.Context, row *orderdomain.SubscriptionAuthorization, challenge bool) (result *orderdomain.SubscriptionAuthorization, created bool, err error) {
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock the existing order so distinct challenge IDs cannot bypass the
		// per-order send interval. No network work occurs in this transaction.
		var order orderdomain.Order
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&order, row.OrderID).Error; e != nil {
			return e
		}
		var prior orderdomain.SubscriptionAuthorization
		e := tx.First(&prior, "id = ?", row.ID).Error
		if e == nil {
			if prior.ContextHash != row.ContextHash || prior.OrderID != row.OrderID || prior.BuyerRef != row.BuyerRef || prior.TenantRef != row.TenantRef {
				return ordercontract.ErrAuthorizationConflict
			}
			result = &prior
			return nil
		}
		if !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		if challenge {
			var count int64
			if e = tx.Model(&orderdomain.SubscriptionAuthorization{}).Where("order_id = ? AND tenant_ref = ? AND created_at > ? AND status <> ?", row.OrderID, row.TenantRef, row.CreatedAt.Add(-time.Minute), "send_failed").Count(&count).Error; e != nil {
				return e
			}
			if count > 0 {
				return ordercontract.ErrAuthorizationRateLimited
			}
		}
		var active int64
		if e = tx.Model(&orderdomain.SubscriptionAuthorization{}).Where("order_id = ? AND expires_at > ? AND status <> ?", row.OrderID, row.CreatedAt, "send_failed").Count(&active).Error; e != nil {
			return e
		}
		if active >= 5 {
			return ordercontract.ErrAuthorizationRateLimited
		}
		if e = tx.Create(row).Error; e != nil {
			return e
		}
		copy := *row
		result, created = &copy, true
		return nil
	})
	return
}

func (s *SubscriptionAuthorizationStore) Update(ctx context.Context, id string, fn func(*orderdomain.SubscriptionAuthorization) error) (result *orderdomain.SubscriptionAuthorization, err error) {
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row orderdomain.SubscriptionAuthorization
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "id = ?", id).Error; e != nil {
			if errors.Is(e, gorm.ErrRecordNotFound) {
				return ordercontract.ErrAuthorizationNotFound
			}
			return e
		}
		if e := fn(&row); e != nil {
			return e
		}
		if e := tx.Save(&row).Error; e != nil {
			return e
		}
		result = &row
		return nil
	})
	return
}
