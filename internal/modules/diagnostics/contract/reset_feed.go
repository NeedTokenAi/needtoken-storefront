package diagnosticscontract

import (
	"context"
	"errors"
	"time"
)

var ErrResetFeedUnavailable = errors.New("reset feed: source unavailable")

type ResetSource struct {
	Type   string `json:"type"`
	Author string `json:"author"`
	URL    string `json:"url"`
}

type ResetRecord struct {
	ID          string      `json:"id"`
	Type        string      `json:"type"`
	AnnouncedAt time.Time   `json:"announced_at"`
	Text        string      `json:"text"`
	TextZH      string      `json:"text_zh,omitempty"`
	Scope       string      `json:"scope,omitempty"`
	ScopeZH     string      `json:"scope_zh,omitempty"`
	Source      ResetSource `json:"source"`
}

// ResetFeedPage is one upstream page, not a complete historical inventory.
type ResetFeedPage struct {
	HasMore bool
	Records []ResetRecord
}

type ResetFeedSnapshot struct {
	SourceURL string        `json:"source_url"`
	FetchedAt time.Time     `json:"fetched_at"`
	Stale     bool          `json:"stale"`
	HasMore   bool          `json:"has_more"`
	Records   []ResetRecord `json:"records"`
}

type ResetFeedSource interface {
	Fetch(context.Context) (ResetFeedPage, error)
}

type ResetFeedReader interface {
	Latest(context.Context) (ResetFeedSnapshot, error)
}

type ChannelResetFeedReader interface {
	ResetFeedReader
	LatestForChannel(context.Context, string) (ResetFeedSnapshot, error)
}
