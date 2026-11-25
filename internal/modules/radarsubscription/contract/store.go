package radarsubscriptioncontract

import (
	"errors"
	"time"

	radarsubscriptiondomain "github.com/dujiao-next/internal/modules/radarsubscription/domain"
	"github.com/dujiao-next/internal/shared/mailbrand"
)

var (
	ErrInvalidChannel       = errors.New("radar subscription: invalid channel")
	ErrEmailBindingRequired = errors.New("radar subscription: verified real email binding required")
	ErrUserNotFound         = errors.New("radar subscription: user not found")
)

// Store persists member radar notification preferences, seen event baselines, and email deliveries.
type Store interface {
	ListByUser(userID uint) ([]radarsubscriptiondomain.Subscription, error)
	UpsertUserChannels(userID uint, channels map[string]bool) error
	ListSubscribedUserIDs(channel string) ([]uint, error)
	CountSeenEvents(channel string) (int64, error)
	GetSeenRecordIDs(channel string, recordIDs []string) (map[string]bool, error)
	MarkEventsSeen(events []radarsubscriptiondomain.SeenEvent) error
	HasSentDelivery(channel, recordID string, userID uint) (bool, error)
	UpsertDelivery(delivery *radarsubscriptiondomain.EventDelivery) error
}

// EmailSender sends reset notification emails to subscribed members.
type EmailSender interface {
	SendRadarResetEmail(toEmail, locale, channel, recordType, scope, summary, sourceURL string, announcedAt time.Time, brand mailbrand.Brand) error
}
