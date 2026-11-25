package radarsubscriptionapp_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dujiao-next/internal/constants"
	diagnosticscontract "github.com/dujiao-next/internal/modules/diagnostics/contract"
	usercontract "github.com/dujiao-next/internal/modules/identity/user/contract"
	userdomain "github.com/dujiao-next/internal/modules/identity/user/domain"
	radarsubscriptionapp "github.com/dujiao-next/internal/modules/radarsubscription/application"
	radarsubscriptioncontract "github.com/dujiao-next/internal/modules/radarsubscription/contract"
	radarsubscriptiondomain "github.com/dujiao-next/internal/modules/radarsubscription/domain"
	"github.com/dujiao-next/internal/shared/mailbrand"
)

type fakeRadarStore struct {
	subs       map[uint]map[string]bool
	seen       map[string]map[string]bool
	deliveries map[string]radarsubscriptiondomain.EventDelivery
}

func newFakeRadarStore() *fakeRadarStore {
	return &fakeRadarStore{
		subs:       make(map[uint]map[string]bool),
		seen:       make(map[string]map[string]bool),
		deliveries: make(map[string]radarsubscriptiondomain.EventDelivery),
	}
}

func (f *fakeRadarStore) ListByUser(userID uint) ([]radarsubscriptiondomain.Subscription, error) {
	var out []radarsubscriptiondomain.Subscription
	for ch, enabled := range f.subs[userID] {
		out = append(out, radarsubscriptiondomain.Subscription{UserID: userID, Channel: ch, Enabled: enabled})
	}
	return out, nil
}

func (f *fakeRadarStore) UpsertUserChannels(userID uint, channels map[string]bool) error {
	if f.subs[userID] == nil {
		f.subs[userID] = make(map[string]bool)
	}
	for ch, enabled := range channels {
		f.subs[userID][ch] = enabled
	}
	return nil
}

func (f *fakeRadarStore) ListSubscribedUserIDs(channel string) ([]uint, error) {
	var ids []uint
	for uid, chs := range f.subs {
		if chs[channel] {
			ids = append(ids, uid)
		}
	}
	return ids, nil
}

func (f *fakeRadarStore) CountSeenEvents(channel string) (int64, error) {
	return int64(len(f.seen[channel])), nil
}

func (f *fakeRadarStore) GetSeenRecordIDs(channel string, recordIDs []string) (map[string]bool, error) {
	out := make(map[string]bool)
	for _, id := range recordIDs {
		if f.seen[channel][id] {
			out[id] = true
		}
	}
	return out, nil
}

func (f *fakeRadarStore) MarkEventsSeen(events []radarsubscriptiondomain.SeenEvent) error {
	for _, e := range events {
		if f.seen[e.Channel] == nil {
			f.seen[e.Channel] = make(map[string]bool)
		}
		f.seen[e.Channel][e.RecordID] = true
	}
	return nil
}

func deliveryKey(channel, recordID string, userID uint) string {
	return channel + "|" + recordID + "|" + string(rune(userID+'0'))
}

func (f *fakeRadarStore) HasSentDelivery(channel, recordID string, userID uint) (bool, error) {
	d, ok := f.deliveries[deliveryKey(channel, recordID, userID)]
	return ok && d.Status == radarsubscriptiondomain.DeliveryStatusSent, nil
}

func (f *fakeRadarStore) UpsertDelivery(delivery *radarsubscriptiondomain.EventDelivery) error {
	f.deliveries[deliveryKey(delivery.Channel, delivery.RecordID, delivery.UserID)] = *delivery
	return nil
}

type fakeUserStore struct {
	usercontract.Store
	users map[uint]*userdomain.User
}

func (f *fakeUserStore) GetByID(id uint) (*userdomain.User, error) {
	return f.users[id], nil
}

func (f *fakeUserStore) ListByIDs(ids []uint) ([]userdomain.User, error) {
	var out []userdomain.User
	for _, id := range ids {
		if u, ok := f.users[id]; ok && u != nil {
			out = append(out, *u)
		}
	}
	return out, nil
}

type fakeFeed struct {
	snapshots map[string]diagnosticscontract.ResetFeedSnapshot
}

func (f *fakeFeed) Latest(ctx context.Context) (diagnosticscontract.ResetFeedSnapshot, error) {
	return f.LatestForChannel(ctx, "codex")
}

func (f *fakeFeed) LatestForChannel(_ context.Context, channel string) (diagnosticscontract.ResetFeedSnapshot, error) {
	return f.snapshots[channel], nil
}

type sentMail struct {
	to, locale, channel, recordType, summary string
}

type fakeMailer struct {
	sent []sentMail
}

func (f *fakeMailer) SendRadarResetEmail(toEmail, locale, channel, recordType, _, summary, _ string, _ time.Time, _ mailbrand.Brand) error {
	f.sent = append(f.sent, sentMail{
		to: toEmail, locale: locale, channel: channel, recordType: recordType, summary: summary,
	})
	return nil
}

func TestRadarSubscriptionLifecycleAndNotifications(t *testing.T) {
	now := time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC)
	store := newFakeRadarStore()
	users := &fakeUserStore{
		users: map[uint]*userdomain.User{
			1: {ID: 1, Email: "alice@example.com", Locale: constants.LocaleZhCN, Status: constants.UserStatusActive},
			2: {ID: 2, Email: "telegram_999@login.local", Locale: constants.LocaleZhCN, Status: constants.UserStatusActive},
			3: {ID: 3, Email: "disabled@example.com", Locale: constants.LocaleEnUS, Status: constants.UserStatusDisabled},
		},
	}
	feed := &fakeFeed{
		snapshots: map[string]diagnosticscontract.ResetFeedSnapshot{
			"claude": {
				Records: []diagnosticscontract.ResetRecord{
					{ID: "claude-old-1", Type: "regular", AnnouncedAt: now.Add(-48 * time.Hour), Text: "Old reset", TextZH: "旧重置"},
				},
			},
		},
	}
	mailer := &fakeMailer{}
	svc := radarsubscriptionapp.NewService(radarsubscriptionapp.Options{
		Store:       store,
		UserStore:   users,
		Feed:        feed,
		EmailSender: mailer,
		Now:         func() time.Time { return now },
	})

	// 1. Placeholder Telegram email must be rejected when enabling subscriptions.
	if _, err := svc.UpdateUserPreferences(2, map[string]bool{"claude": true}); !errors.Is(err, radarsubscriptioncontract.ErrEmailBindingRequired) {
		t.Fatalf("expected ErrEmailBindingRequired for telegram placeholder email, got %v", err)
	}

	// 2. Invalid channel rejected.
	if _, err := svc.UpdateUserPreferences(1, map[string]bool{"unknown": true}); !errors.Is(err, radarsubscriptioncontract.ErrInvalidChannel) {
		t.Fatalf("expected ErrInvalidChannel, got %v", err)
	}

	// 3. Valid member subscribes to Claude.
	prefs, err := svc.UpdateUserPreferences(1, map[string]bool{"claude": true})
	if err != nil {
		t.Fatalf("subscribe claude failed: %v", err)
	}
	if !prefs.EmailBound || prefs.Email != "alice@example.com" || !prefs.Channels["claude"] || prefs.SubscribedCount != 1 {
		t.Fatalf("unexpected prefs: %+v", prefs)
	}
	// Disabled user manually in store must be skipped during notify.
	_ = store.UpsertUserChannels(3, map[string]bool{"claude": true})

	// 4. Cold start poll seeds baseline without emailing historical announcements.
	if err := svc.PollAndNotify(context.Background()); err != nil {
		t.Fatalf("cold start poll failed: %v", err)
	}
	if len(mailer.sent) != 0 {
		t.Fatalf("cold start should not send historical emails, got %d", len(mailer.sent))
	}

	// 5. New recent reset arrives -> only active subscribed member receives email.
	feed.snapshots["claude"] = diagnosticscontract.ResetFeedSnapshot{
		Records: []diagnosticscontract.ResetRecord{
			{ID: "claude-new-2", Type: "banked", AnnouncedAt: now.Add(-10 * time.Minute), Text: "Reset card issued", TextZH: "发放重置卡"},
			{ID: "claude-old-1", Type: "regular", AnnouncedAt: now.Add(-48 * time.Hour), Text: "Old reset", TextZH: "旧重置"},
		},
	}
	if err := svc.PollAndNotify(context.Background()); err != nil {
		t.Fatalf("second poll failed: %v", err)
	}
	if len(mailer.sent) != 1 {
		t.Fatalf("expected 1 email sent, got %d", len(mailer.sent))
	}
	if mailer.sent[0].to != "alice@example.com" || mailer.sent[0].channel != "claude" || mailer.sent[0].summary != "发放重置卡" {
		t.Fatalf("unexpected email payload: %+v", mailer.sent[0])
	}

	// 6. Polling again is strictly idempotent.
	if err := svc.PollAndNotify(context.Background()); err != nil {
		t.Fatalf("third poll failed: %v", err)
	}
	if len(mailer.sent) != 1 {
		t.Fatalf("duplicate notification sent on repeat poll: %d", len(mailer.sent))
	}
}
