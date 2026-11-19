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
	statusHistoryWait  = 3 * time.Second
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
	sourceURL := "https://status.openai.com/"
	if provider == diagnosticscontract.ServiceStatusClaude {
		sourceURL = "https://status.claude.com/"
	}
	history := diagnosticscontract.ServiceStatusHistory{State: "unavailable", Coverage: "partial", SourceURL: sourceURL + "history", Incidents: []diagnosticscontract.ServiceStatusIncident{}}
	var series []diagnosticscontract.ServiceStatusSeries
	var err error
	if s.source == nil {
		err = diagnosticscontract.ErrServiceStatusUnavailable
	} else {
		summary, err = s.source.Fetch(ctx, provider)
	}
	if err == nil {
		historySource, hasHistory := s.source.(diagnosticscontract.ServiceStatusHistorySource)
		seriesSource, hasSeries := s.source.(diagnosticscontract.ServiceStatusSeriesSource)
		if hasHistory || hasSeries {
			budget := statusHistoryBudget(ctx)
			var wg sync.WaitGroup
			if hasHistory {
				wg.Add(1)
				go func() {
					defer wg.Done()
					historyCtx, historyCancel := context.WithTimeout(ctx, budget)
					recent, historyErr := historySource.FetchHistory(historyCtx, provider)
					historyCancel()
					if historyErr == nil && recent.State == "ready" {
						// Recent announcements never establish a complete uptime window.
						history.State = "ready"
						history.Incidents = recent.Incidents
					}
				}()
			}
			if hasSeries {
				wg.Add(1)
				go func() {
					defer wg.Done()
					seriesCtx, seriesCancel := context.WithTimeout(ctx, budget)
					fetched, seriesErr := seriesSource.FetchSeries(seriesCtx, provider, summary)
					seriesCancel()
					if seriesErr == nil && len(fetched) > 0 {
						series = fetched
					}
				}()
			}
			wg.Wait()
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now().UTC()
	if err == nil {
		result := copyStatusSnapshot(diagnosticscontract.ServiceStatusSnapshot{Provider: provider, SourceURL: sourceURL, FetchedAt: now, Summary: summary, History: history, Series: series}, false)
		entry.latest = &result
		entry.retry = time.Time{}
	} else {
		entry.retry = now.Add(statusRetryAfter)
	}
	entry.flight = nil
	close(done)
}

func statusHistoryBudget(ctx context.Context) time.Duration {
	budget := statusHistoryWait
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return 0
		}
		// Optional history must leave time to return a valid current summary;
		// otherwise it can consume the browser's entire status request timeout.
		if remaining/2 < budget {
			budget = remaining / 2
		}
	}
	return budget
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
	value.Summary.Incidents = copyStatusIncidents(value.Summary.Incidents)
	value.History.Incidents = copyStatusIncidents(value.History.Incidents)
	value.Series = copyStatusSeries(value.Series)
	return value
}

func copyStatusSeries(rows []diagnosticscontract.ServiceStatusSeries) []diagnosticscontract.ServiceStatusSeries {
	if len(rows) == 0 {
		return nil
	}
	result := append(make([]diagnosticscontract.ServiceStatusSeries, 0, len(rows)), rows...)
	for index := range result {
		item := &result[index]
		if item.UptimePercent != nil {
			percent := *item.UptimePercent
			item.UptimePercent = &percent
		}
		item.Days = append(make([]diagnosticscontract.ServiceStatusSeriesDay, 0, len(item.Days)), item.Days...)
		for dayIndex := range item.Days {
			if len(item.Days[dayIndex].Events) > 0 {
				item.Days[dayIndex].Events = append(make([]diagnosticscontract.ServiceStatusSeriesEvent, 0, len(item.Days[dayIndex].Events)), item.Days[dayIndex].Events...)
			}
		}
	}
	return result
}

func copyStatusIncidents(rows []diagnosticscontract.ServiceStatusIncident) []diagnosticscontract.ServiceStatusIncident {
	result := append(make([]diagnosticscontract.ServiceStatusIncident, 0, len(rows)), rows...)
	for index := range result {
		incident := &result[index]
		incident.CreatedAt = copyStatusTime(incident.CreatedAt)
		incident.StartedAt = copyStatusTime(incident.StartedAt)
		incident.UpdatedAt = copyStatusTime(incident.UpdatedAt)
		incident.ResolvedAt = copyStatusTime(incident.ResolvedAt)
		incident.Components = append(make([]diagnosticscontract.ServiceStatusIncidentComponent, 0, len(incident.Components)), incident.Components...)
		incident.IncidentUpdates = append(make([]diagnosticscontract.ServiceStatusIncidentUpdate, 0, len(incident.IncidentUpdates)), incident.IncidentUpdates...)
		for updateIndex := range incident.IncidentUpdates {
			update := &incident.IncidentUpdates[updateIndex]
			update.DisplayAt = copyStatusTime(update.DisplayAt)
			update.CreatedAt = copyStatusTime(update.CreatedAt)
			update.UpdatedAt = copyStatusTime(update.UpdatedAt)
			update.AffectedComponents = append(make([]diagnosticscontract.ServiceStatusAffectedComponent, 0, len(update.AffectedComponents)), update.AffectedComponents...)
		}
	}
	return result
}

func copyStatusTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

var _ diagnosticscontract.ServiceStatusReader = (*ServiceStatus)(nil)
