package radarsubscriptiondomain

import (
	"strings"
	"time"
)

const (
	ChannelCodex  = "codex"
	ChannelClaude = "claude"
	ChannelGemini = "gemini"
	ChannelGrok   = "grok"

	DeliveryStatusSent   = "sent"
	DeliveryStatusFailed = "failed"
)

// ValidChannels lists all supported reset radar channels in display order.
var ValidChannels = []string{
	ChannelCodex,
	ChannelClaude,
	ChannelGemini,
	ChannelGrok,
}

// NormalizeChannel validates and normalizes a radar channel key.
func NormalizeChannel(raw string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case ChannelCodex:
		return ChannelCodex, true
	case ChannelClaude:
		return ChannelClaude, true
	case ChannelGemini:
		return ChannelGemini, true
	case ChannelGrok:
		return ChannelGrok, true
	default:
		return "", false
	}
}

// Subscription stores a member's email notification preference for one radar channel.
type Subscription struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_radar_sub_user_channel;index"`
	Channel   string    `gorm:"size:32;not null;uniqueIndex:idx_radar_sub_user_channel;index:idx_radar_sub_channel_enabled"`
	Enabled   bool      `gorm:"not null;index:idx_radar_sub_channel_enabled"`
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time
}

func (Subscription) TableName() string {
	return "radar_subscriptions"
}

// SeenEvent records each observed upstream reset announcement so restarts do not re-trigger notifications.
type SeenEvent struct {
	ID              uint      `gorm:"primaryKey"`
	Channel         string    `gorm:"size:32;not null;uniqueIndex:idx_radar_seen_channel_record"`
	RecordID        string    `gorm:"size:191;not null;uniqueIndex:idx_radar_seen_channel_record"`
	AnnouncedAt     time.Time `gorm:"index"`
	FirstSeenAt     time.Time `gorm:"index"`
	InitializedOnly bool      `gorm:"not null"`
}

func (SeenEvent) TableName() string {
	return "radar_seen_events"
}

// EventDelivery tracks per-user email delivery idempotency for a specific reset event.
type EventDelivery struct {
	ID        uint       `gorm:"primaryKey"`
	Channel   string     `gorm:"size:32;not null;uniqueIndex:idx_radar_delivery_unique"`
	RecordID  string     `gorm:"size:191;not null;uniqueIndex:idx_radar_delivery_unique"`
	UserID    uint       `gorm:"not null;uniqueIndex:idx_radar_delivery_unique;index"`
	Email     string     `gorm:"size:255;not null"`
	Status    string     `gorm:"size:32;not null;index"`
	Attempts  int        `gorm:"not null"`
	LastError string     `gorm:"size:512"`
	SentAt    *time.Time `gorm:"index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (EventDelivery) TableName() string {
	return "radar_event_deliveries"
}
