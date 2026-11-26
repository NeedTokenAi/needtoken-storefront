package radarsubscriptionapp

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/logger"
	diagnosticscontract "github.com/dujiao-next/internal/modules/diagnostics/contract"
	usercontract "github.com/dujiao-next/internal/modules/identity/user/contract"
	radarsubscriptioncontract "github.com/dujiao-next/internal/modules/radarsubscription/contract"
	radarsubscriptiondomain "github.com/dujiao-next/internal/modules/radarsubscription/domain"
	"github.com/dujiao-next/internal/shared/mailbrand"
	"github.com/dujiao-next/internal/telegramidentity"
)

const (
	recentEventWindow  = 24 * time.Hour
	perChannelFetchTTL = 12 * time.Second
)

// UserPreferences describes a member's current radar subscription state.
type UserPreferences struct {
	Email           string          `json:"email"`
	EmailBound      bool            `json:"email_bound"`
	Channels        map[string]bool `json:"channels"`
	SubscribedCount int             `json:"subscribed_count"`
}

// Service manages member radar subscriptions and dispatches reset notification emails.
type Service struct {
	store         radarsubscriptioncontract.Store
	userStore     usercontract.Store
	feed          diagnosticscontract.ChannelResetFeedReader
	emailSender   radarsubscriptioncontract.EmailSender
	brandResolver mailbrand.Resolver
	now           func() time.Time
	pollMu        sync.Mutex
}

// Options configures a radar subscription service.
type Options struct {
	Store         radarsubscriptioncontract.Store
	UserStore     usercontract.Store
	Feed          diagnosticscontract.ChannelResetFeedReader
	EmailSender   radarsubscriptioncontract.EmailSender
	BrandResolver mailbrand.Resolver
	Now           func() time.Time
}

func NewService(opts Options) *Service {
	nowFn := opts.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	return &Service{
		store:         opts.Store,
		userStore:     opts.UserStore,
		feed:          opts.Feed,
		emailSender:   opts.EmailSender,
		brandResolver: opts.BrandResolver,
		now:           nowFn,
	}
}

func (s *Service) GetUserPreferences(userID uint) (*UserPreferences, error) {
	user, err := s.userStore.GetByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, radarsubscriptioncontract.ErrUserNotFound
	}
	rows, err := s.store.ListByUser(userID)
	if err != nil {
		return nil, err
	}
	channels := make(map[string]bool, len(radarsubscriptiondomain.ValidChannels))
	for _, ch := range radarsubscriptiondomain.ValidChannels {
		channels[ch] = false
	}
	count := 0
	for _, row := range rows {
		if ch, ok := radarsubscriptiondomain.NormalizeChannel(row.Channel); ok {
			channels[ch] = row.Enabled
			if row.Enabled {
				count++
			}
		}
	}
	email := strings.TrimSpace(user.Email)
	bound := email != "" && !telegramidentity.IsPlaceholderEmail(email)
	displayEmail := ""
	if bound {
		displayEmail = email
	}
	return &UserPreferences{
		Email:           displayEmail,
		EmailBound:      bound,
		Channels:        channels,
		SubscribedCount: count,
	}, nil
}

func (s *Service) UpdateUserPreferences(userID uint, rawChannels map[string]bool) (*UserPreferences, error) {
	if len(rawChannels) == 0 {
		return nil, radarsubscriptioncontract.ErrInvalidChannel
	}
	normalized := make(map[string]bool, len(rawChannels))
	enablingAny := false
	for rawCh, enabled := range rawChannels {
		ch, ok := radarsubscriptiondomain.NormalizeChannel(rawCh)
		if !ok {
			return nil, radarsubscriptioncontract.ErrInvalidChannel
		}
		normalized[ch] = enabled
		if enabled {
			enablingAny = true
		}
	}

	user, err := s.userStore.GetByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, radarsubscriptioncontract.ErrUserNotFound
	}
	email := strings.TrimSpace(user.Email)
	if enablingAny && (email == "" || telegramidentity.IsPlaceholderEmail(email)) {
		return nil, radarsubscriptioncontract.ErrEmailBindingRequired
	}

	if err := s.store.UpsertUserChannels(userID, normalized); err != nil {
		return nil, err
	}
	return s.GetUserPreferences(userID)
}

// PollAndNotify checks all radar channels for newly observed reset announcements
// and sends localized notification emails to subscribed active members.
// Network I/O and SMTP delivery always occur outside database transactions.
func (s *Service) PollAndNotify(ctx context.Context) error {
	if s == nil || s.store == nil || s.feed == nil {
		return nil
	}
	if !s.pollMu.TryLock() {
		return nil
	}
	defer s.pollMu.Unlock()

	var firstErr error
	for _, channel := range radarsubscriptiondomain.ValidChannels {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := s.pollChannel(ctx, channel); err != nil {
			logger.Warnw("radar_notify_poll_channel_failed", "channel", channel, "error", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (s *Service) pollChannel(ctx context.Context, channel string) error {
	fetchCtx, cancel := context.WithTimeout(ctx, perChannelFetchTTL)
	defer cancel()

	snapshot, err := s.feed.LatestForChannel(fetchCtx, channel)
	if err != nil {
		return err
	}
	if len(snapshot.Records) == 0 {
		return nil
	}

	now := s.now().UTC()
	seenCount, err := s.store.CountSeenEvents(channel)
	if err != nil {
		return err
	}

	// Cold start: seed baseline without flooding historical announcements.
	if seenCount == 0 {
		baseline := make([]radarsubscriptiondomain.SeenEvent, 0, len(snapshot.Records))
		for _, rec := range snapshot.Records {
			id := strings.TrimSpace(rec.ID)
			if id == "" {
				continue
			}
			baseline = append(baseline, radarsubscriptiondomain.SeenEvent{
				Channel:         channel,
				RecordID:        id,
				AnnouncedAt:     rec.AnnouncedAt.UTC(),
				FirstSeenAt:     now,
				InitializedOnly: true,
			})
		}
		return s.store.MarkEventsSeen(baseline)
	}

	recordIDs := make([]string, 0, len(snapshot.Records))
	for _, rec := range snapshot.Records {
		if id := strings.TrimSpace(rec.ID); id != "" {
			recordIDs = append(recordIDs, id)
		}
	}
	alreadySeen, err := s.store.GetSeenRecordIDs(channel, recordIDs)
	if err != nil {
		return err
	}

	newRecords := make([]diagnosticscontract.ResetRecord, 0)
	toPersist := make([]radarsubscriptiondomain.SeenEvent, 0)
	for _, rec := range snapshot.Records {
		id := strings.TrimSpace(rec.ID)
		if id == "" || alreadySeen[id] {
			continue
		}
		alreadySeen[id] = true
		toPersist = append(toPersist, radarsubscriptiondomain.SeenEvent{
			Channel:         channel,
			RecordID:        id,
			AnnouncedAt:     rec.AnnouncedAt.UTC(),
			FirstSeenAt:     now,
			InitializedOnly: false,
		})
		if !rec.AnnouncedAt.IsZero() && now.Sub(rec.AnnouncedAt.UTC()) > recentEventWindow {
			continue
		}
		newRecords = append(newRecords, rec)
	}
	if len(toPersist) > 0 {
		if err := s.store.MarkEventsSeen(toPersist); err != nil {
			return err
		}
	}
	if len(newRecords) == 0 || s.emailSender == nil || s.userStore == nil {
		return nil
	}

	userIDs, err := s.store.ListSubscribedUserIDs(channel)
	if err != nil || len(userIDs) == 0 {
		return err
	}
	excludedUsers, err := s.store.ListExcludedUserIDs(userIDs)
	if err != nil {
		return err
	}
	if len(excludedUsers) > 0 {
		filtered := make([]uint, 0, len(userIDs))
		for _, uid := range userIDs {
			if !excludedUsers[uid] {
				filtered = append(filtered, uid)
			}
		}
		userIDs = filtered
	}
	if len(userIDs) == 0 {
		return nil
	}
	users, err := s.userStore.ListByIDs(userIDs)
	if err != nil || len(users) == 0 {
		return err
	}

	var brand mailbrand.Brand
	if s.brandResolver != nil {
		if resolved, brandErr := s.brandResolver.ResolveEmailBrand(ctx, mailbrand.Scope{}); brandErr == nil {
			brand = resolved
		}
	}

	for _, rec := range newRecords {
		recID := strings.TrimSpace(rec.ID)
		for _, user := range users {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			email := strings.TrimSpace(user.Email)
			if !strings.EqualFold(strings.TrimSpace(user.Status), constants.UserStatusActive) ||
				email == "" || telegramidentity.IsPlaceholderEmail(email) {
				continue
			}
			sent, checkErr := s.store.HasSentDelivery(channel, recID, user.ID)
			if checkErr != nil || sent {
				continue
			}

			locale := strings.TrimSpace(user.Locale)
			summary := strings.TrimSpace(rec.Text)
			scope := strings.TrimSpace(rec.Scope)
			if strings.HasPrefix(strings.ToLower(locale), "zh") {
				if strings.TrimSpace(rec.TextZH) != "" {
					summary = strings.TrimSpace(rec.TextZH)
				}
				if strings.TrimSpace(rec.ScopeZH) != "" {
					scope = strings.TrimSpace(rec.ScopeZH)
				}
			}

			sendErr := s.emailSender.SendRadarResetEmail(
				email,
				locale,
				channel,
				strings.TrimSpace(rec.Type),
				scope,
				summary,
				strings.TrimSpace(rec.Source.URL),
				rec.AnnouncedAt.UTC(),
				brand,
			)
			delivery := &radarsubscriptiondomain.EventDelivery{
				Channel:  channel,
				RecordID: recID,
				UserID:   user.ID,
				Email:    email,
				Attempts: 1,
			}
			if sendErr != nil {
				delivery.Status = radarsubscriptiondomain.DeliveryStatusFailed
				delivery.LastError = truncateString(sendErr.Error(), 500)
				logger.Warnw("radar_notify_email_failed", "channel", channel, "record_id", recID, "user_id", user.ID, "error", sendErr)
			} else {
				sentAt := s.now().UTC()
				delivery.Status = radarsubscriptiondomain.DeliveryStatusSent
				delivery.SentAt = &sentAt
			}
			if recordErr := s.store.UpsertDelivery(delivery); recordErr != nil {
				logger.Warnw("radar_notify_delivery_record_failed", "channel", channel, "record_id", recID, "user_id", user.ID, "error", recordErr)
			}
		}
	}
	return nil
}

func truncateString(v string, maxLen int) string {
	v = strings.TrimSpace(v)
	if len(v) <= maxLen {
		return v
	}
	return v[:maxLen]
}

// IsInvalidChannelError reports whether err is ErrInvalidChannel.
func IsInvalidChannelError(err error) bool {
	return errors.Is(err, radarsubscriptioncontract.ErrInvalidChannel)
}
