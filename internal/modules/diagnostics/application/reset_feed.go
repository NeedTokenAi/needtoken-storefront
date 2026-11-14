package diagnosticsapplication

import (
	"context"
	"sync"
	"time"

	diagnosticscontract "github.com/dujiao-next/internal/modules/diagnostics/contract"
)

const (
	resetFeedSourceURL = "https://codex-resets.com/"
	resetFreshFor      = 60 * time.Second
	resetRetryAfter    = 30 * time.Second
	resetStaleFor      = 12 * time.Hour
	resetFetchTimeout  = 8 * time.Second
)

// ResetFeed keeps one public page in memory; no account or quota data is stored.
type ResetFeed struct {
	source diagnosticscontract.ResetFeedSource
	now    func() time.Time
	mu     sync.Mutex
	latest *diagnosticscontract.ResetFeedSnapshot
	retry  time.Time
	flight chan struct{}
}

func NewResetFeed(source diagnosticscontract.ResetFeedSource) *ResetFeed {
	return &ResetFeed{source: source, now: time.Now}
}

func (s *ResetFeed) Latest(ctx context.Context) (diagnosticscontract.ResetFeedSnapshot, error) {
	for {
		if err := ctx.Err(); err != nil {
			return diagnosticscontract.ResetFeedSnapshot{}, err
		}
		s.mu.Lock()
		now := s.now()
		if s.latest != nil && now.Sub(s.latest.FetchedAt) < resetFreshFor {
			result := copySnapshot(*s.latest, false)
			s.mu.Unlock()
			return result, nil
		}
		done := s.flight
		if done == nil && now.Before(s.retry) {
			result, err := s.staleLocked(now)
			s.mu.Unlock()
			return result, err
		}
		if done == nil {
			done = make(chan struct{})
			s.flight = done
			go s.refresh(done)
		}
		s.mu.Unlock()
		select {
		case <-ctx.Done():
			return diagnosticscontract.ResetFeedSnapshot{}, ctx.Err()
		case <-done:
		}
	}
}

func (s *ResetFeed) refresh(done chan struct{}) {
	// One caller leaving must not cancel a refresh shared by other visitors.
	ctx, cancel := context.WithTimeout(context.Background(), resetFetchTimeout)
	defer cancel()
	var page diagnosticscontract.ResetFeedPage
	var err error
	if s.source == nil {
		err = diagnosticscontract.ErrResetFeedUnavailable
	} else {
		page, err = s.source.Fetch(ctx)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now().UTC()
	if err == nil {
		result := diagnosticscontract.ResetFeedSnapshot{SourceURL: resetFeedSourceURL, FetchedAt: now, HasMore: page.HasMore, Records: page.Records}
		result = copySnapshot(result, false)
		s.latest = &result
		s.retry = time.Time{}
	} else {
		s.retry = now.Add(resetRetryAfter)
	}
	s.flight = nil
	close(done)
}

func (s *ResetFeed) staleLocked(now time.Time) (diagnosticscontract.ResetFeedSnapshot, error) {
	if s.latest != nil && now.Sub(s.latest.FetchedAt) <= resetStaleFor {
		return copySnapshot(*s.latest, true), nil
	}
	return diagnosticscontract.ResetFeedSnapshot{}, diagnosticscontract.ErrResetFeedUnavailable
}

func copySnapshot(value diagnosticscontract.ResetFeedSnapshot, stale bool) diagnosticscontract.ResetFeedSnapshot {
	value.Stale = stale
	value.Records = append(make([]diagnosticscontract.ResetRecord, 0, len(value.Records)), value.Records...)
	return value
}

var _ diagnosticscontract.ResetFeedReader = (*ResetFeed)(nil)
