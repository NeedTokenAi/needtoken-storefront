package gormstore

import (
	"time"

	radarsubscriptioncontract "github.com/dujiao-next/internal/modules/radarsubscription/contract"
	radarsubscriptiondomain "github.com/dujiao-next/internal/modules/radarsubscription/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Store struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) ListByUser(userID uint) ([]radarsubscriptiondomain.Subscription, error) {
	var rows []radarsubscriptiondomain.Subscription
	if err := s.db.Where("user_id = ?", userID).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Store) ListAllSubscriptions() ([]radarsubscriptiondomain.Subscription, error) {
	var rows []radarsubscriptiondomain.Subscription
	if err := s.db.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Store) UpsertUserChannels(userID uint, channels map[string]bool) error {
	if len(channels) == 0 {
		return nil
	}
	now := time.Now().UTC()
	return s.db.Transaction(func(tx *gorm.DB) error {
		for channel, enabled := range channels {
			row := radarsubscriptiondomain.Subscription{
				UserID:    userID,
				Channel:   channel,
				Enabled:   enabled,
				CreatedAt: now,
				UpdatedAt: now,
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "user_id"}, {Name: "channel"}},
				DoUpdates: clause.Assignments(map[string]any{
					"enabled":    enabled,
					"updated_at": now,
				}),
			}).Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) ListSubscribedUserIDs(channel string) ([]uint, error) {
	var userIDs []uint
	if err := s.db.Model(&radarsubscriptiondomain.Subscription{}).
		Where("channel = ? AND enabled = ?", channel, true).
		Pluck("user_id", &userIDs).Error; err != nil {
		return nil, err
	}
	return userIDs, nil
}

func (s *Store) ListExcludedUserIDs(userIDs []uint) (map[uint]bool, error) {
	excluded := make(map[uint]bool)
	if len(userIDs) == 0 {
		return excluded, nil
	}
	var ids []uint
	if err := s.db.Model(&radarsubscriptiondomain.MemberExclusion{}).
		Where("user_id IN ? AND excluded = ?", userIDs, true).
		Pluck("user_id", &ids).Error; err != nil {
		return nil, err
	}
	for _, id := range ids {
		excluded[id] = true
	}
	return excluded, nil
}

func (s *Store) ListAllExclusions() ([]radarsubscriptiondomain.MemberExclusion, error) {
	var rows []radarsubscriptiondomain.MemberExclusion
	if err := s.db.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Store) UpsertMemberExclusion(userID uint, excluded bool, reason, updatedBy string) (*radarsubscriptiondomain.MemberExclusion, error) {
	now := time.Now().UTC()
	row := radarsubscriptiondomain.MemberExclusion{
		UserID:    userID,
		Excluded:  excluded,
		Reason:    reason,
		UpdatedBy: updatedBy,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"excluded":   excluded,
			"reason":     reason,
			"updated_by": updatedBy,
			"updated_at": now,
		}),
	}).Create(&row).Error; err != nil {
		return nil, err
	}
	var persisted radarsubscriptiondomain.MemberExclusion
	if err := s.db.Where("user_id = ?", userID).First(&persisted).Error; err != nil {
		return nil, err
	}
	return &persisted, nil
}

func (s *Store) CountSeenEvents(channel string) (int64, error) {
	var count int64
	if err := s.db.Model(&radarsubscriptiondomain.SeenEvent{}).
		Where("channel = ?", channel).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Store) GetSeenRecordIDs(channel string, recordIDs []string) (map[string]bool, error) {
	seen := make(map[string]bool, len(recordIDs))
	if len(recordIDs) == 0 {
		return seen, nil
	}
	var existing []string
	if err := s.db.Model(&radarsubscriptiondomain.SeenEvent{}).
		Where("channel = ? AND record_id IN ?", channel, recordIDs).
		Pluck("record_id", &existing).Error; err != nil {
		return nil, err
	}
	for _, id := range existing {
		seen[id] = true
	}
	return seen, nil
}

func (s *Store) MarkEventsSeen(events []radarsubscriptiondomain.SeenEvent) error {
	if len(events) == 0 {
		return nil
	}
	return s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "channel"}, {Name: "record_id"}},
		DoNothing: true,
	}).Create(&events).Error
}

func (s *Store) HasSentDelivery(channel, recordID string, userID uint) (bool, error) {
	var count int64
	if err := s.db.Model(&radarsubscriptiondomain.EventDelivery{}).
		Where("channel = ? AND record_id = ? AND user_id = ? AND status = ?", channel, recordID, userID, radarsubscriptiondomain.DeliveryStatusSent).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Store) UpsertDelivery(delivery *radarsubscriptiondomain.EventDelivery) error {
	if delivery == nil {
		return nil
	}
	now := time.Now().UTC()
	if delivery.CreatedAt.IsZero() {
		delivery.CreatedAt = now
	}
	delivery.UpdatedAt = now
	return s.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "channel"}, {Name: "record_id"}, {Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"email":      delivery.Email,
			"status":     delivery.Status,
			"attempts":   gorm.Expr("attempts + 1"),
			"last_error": delivery.LastError,
			"sent_at":    delivery.SentAt,
			"updated_at": now,
		}),
	}).Create(delivery).Error
}

func (s *Store) ListAllDeliveries() ([]radarsubscriptiondomain.EventDelivery, error) {
	var rows []radarsubscriptiondomain.EventDelivery
	if err := s.db.Order("updated_at DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Store) ListUserDeliveries(userID uint, limit int) ([]radarsubscriptiondomain.EventDelivery, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	var rows []radarsubscriptiondomain.EventDelivery
	if err := s.db.Where("user_id = ?", userID).Order("updated_at DESC, id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

var _ radarsubscriptioncontract.Store = (*Store)(nil)
