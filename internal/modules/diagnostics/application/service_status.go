package diagnosticsapplication

import (
	"context"
	"sync"
	"time"

	diagnosticscontract "github.com/dujiao-next/internal/modules/diagnostics/contract"
)

const (
	statusFreshFor     = 60 * time.Second
	statusRetryAfter   = 30 * time.Second
	statusStaleFor     = 10 * time.Minute
	statusFetchTimeout = 8 * time.Second
)

type statusCacheEntry struct {
	latest *diagnosticscontract.ServiceStatusSnapshot
	retry  time.Time
	flight chan struct{}
}

type ServiceStatus struct {
	source  diagnosticscontract.ServiceStatusSource
	now     func() time.Time
	mu      sync.Mutex
	entries map[diagnosticscontract.ServiceStatusProvider]*statusCacheEntry
}

func NewServiceStatus(source diagnosticscontract.ServiceStatusSource) *ServiceStatus {
	return &ServiceStatus{source: source, now: time.Now, entries: make(map[diagnosticscontract.ServiceStatusProvider]*statusCacheEntry)}
}

func (s *ServiceStatus) Latest(ctx context.Context, provider diagnosticscontract.ServiceStatusProvider) (diagnosticscontract.ServiceStatusSnapshot, error) {
	if !diagnosticscontract.ValidServiceStatusProvider(provider) {
		return diagnosticscontract.ServiceStatusSnapshot{}, diagnosticscontract.ErrServiceStatusProvider
	}
	for {
		if err := ctx.Err(); err != nil {
			return diagnosticscontract.ServiceStatusSnapshot{}, err
		}
		s.mu.Lock()
		entry := s.entries[provider]
		if entry == nil {
			entry = &statusCacheEntry{}
			s.entries[provider] = entry
		}
		now := s.now()
		if entry.latest != nil && now.Sub(entry.latest.FetchedAt) >= 0 && now.Sub(entry.latest.FetchedAt) < statusFreshFor {
			result := copyStatusSnapshot(*entry.latest, false)
			s.mu.Unlock()
			return result, nil
		}
		done := entry.flight
		if done == nil && now.Before(entry.retry) {
			result, err := staleStatus(entry, now)
			s.mu.Unlock()
			return result, err
		}
		if done == nil {
			done = make(chan struct{})
			entry.flight = done
			go s.refreshStatus(provider, entry, done)
		}
		s.mu.Unlock()
		select {
		case <-ctx.Done():
			return diagnosticscontract.ServiceStatusSnapshot{}, ctx.Err()
		case <-done:
		}
	}
}

func (s *ServiceStatus) refreshStatus(provider diagnosticscontract.ServiceStatusProvider, entry *statusCacheEntry, done chan struct{}) {
	ctx, cancel := context.WithTimeout(context.Background(), statusFetchTimeout)
	defer cancel()
	var summary diagnosticscontract.ServiceStatusSummary
	var err error
	if s.source == nil {
		err = diagnosticscontract.ErrServiceStatusUnavailable
	} else {
		summary, err = s.source.Fetch(ctx, provider)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now().UTC()
	if err == nil {
		sourceURL := "https://status.openai.com/"
		if provider == diagnosticscontract.ServiceStatusClaude {
			sourceURL = "https://status.claude.com/"
		}
		result := copyStatusSnapshot(diagnosticscontract.ServiceStatusSnapshot{Provider: provider, SourceURL: sourceURL, FetchedAt: now, Summary: summary}, false)
		entry.latest = &result
		entry.retry = time.Time{}
	} else {
		entry.retry = now.Add(statusRetryAfter)
	}
	entry.flight = nil
	close(done)
}

func staleStatus(entry *statusCacheEntry, now time.Time) (diagnosticscontract.ServiceStatusSnapshot, error) {
	if entry.latest != nil {
		age := now.Sub(entry.latest.FetchedAt)
		if age >= 0 && age < statusStaleFor {
			return copyStatusSnapshot(*entry.latest, true), nil
		}
	}
	return diagnosticscontract.ServiceStatusSnapshot{}, diagnosticscontract.ErrServiceStatusUnavailable
}

func copyStatusSnapshot(value diagnosticscontract.ServiceStatusSnapshot, stale bool) diagnosticscontract.ServiceStatusSnapshot {
	value.Stale = stale
	value.Summary.Components = append(make([]diagnosticscontract.ServiceStatusComponent, 0, len(value.Summary.Components)), value.Summary.Components...)
	for index := range value.Summary.Components {
		value.Summary.Components[index].Components = append([]string(nil), value.Summary.Components[index].Components...)
	}
	if value.Summary.Page.UpdatedAt != nil {
		updated := *value.Summary.Page.UpdatedAt
		value.Summary.Page.UpdatedAt = &updated
	}
	return value
}

var _ diagnosticscontract.ServiceStatusReader = (*ServiceStatus)(nil)
