package diagnosticsapplication

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	diagnosticscontract "github.com/dujiao-next/internal/modules/diagnostics/contract"
)

type statusSourceFunc func(context.Context, diagnosticscontract.ServiceStatusProvider) (diagnosticscontract.ServiceStatusSummary, error)

func (f statusSourceFunc) Fetch(ctx context.Context, provider diagnosticscontract.ServiceStatusProvider) (diagnosticscontract.ServiceStatusSummary, error) {
	return f(ctx, provider)
}

func fixtureStatusSummary() diagnosticscontract.ServiceStatusSummary {
	updated := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return diagnosticscontract.ServiceStatusSummary{Page: diagnosticscontract.ServiceStatusPage{Name: "Official", UpdatedAt: &updated}, Components: []diagnosticscontract.ServiceStatusComponent{{ID: "group", Name: "ChatGPT", Group: true, Components: []string{"child"}}}}
}

func TestStatusCacheSeparatesProvidersPreservesSourceTimeAndCopiesNestedValues(t *testing.T) {
	base := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)
	var elapsed atomic.Int64
	var calls atomic.Int32
	service := NewServiceStatus(statusSourceFunc(func(context.Context, diagnosticscontract.ServiceStatusProvider) (diagnosticscontract.ServiceStatusSummary, error) {
		calls.Add(1)
		return fixtureStatusSummary(), nil
	}))
	service.now = func() time.Time { return base.Add(time.Duration(elapsed.Load())) }
	first, err := service.Latest(context.Background(), diagnosticscontract.ServiceStatusOpenAI)
	if err != nil || first.Stale || !first.FetchedAt.Equal(base) || first.Summary.Page.UpdatedAt.Month() != time.January || first.SourceURL != "https://status.openai.com/" {
		t.Fatalf("invalid source/fetch timestamps: %#v %v", first, err)
	}
	first.Summary.Components[0].Components[0] = "caller changed"
	*first.Summary.Page.UpdatedAt = base
	elapsed.Store(int64(59 * time.Second))
	cached, err := service.Latest(context.Background(), diagnosticscontract.ServiceStatusOpenAI)
	if err != nil || calls.Load() != 1 || !cached.FetchedAt.Equal(base) || cached.Summary.Components[0].Components[0] != "child" || cached.Summary.Page.UpdatedAt.Month() != time.January {
		t.Fatal("cache hit changed successful time or retained mutable caller data")
	}
	claude, err := service.Latest(context.Background(), diagnosticscontract.ServiceStatusClaude)
	if err != nil || calls.Load() != 2 || claude.Provider != diagnosticscontract.ServiceStatusClaude || claude.SourceURL != "https://status.claude.com/" {
		t.Fatal("providers shared one cached response")
	}
	if _, err := service.Latest(context.Background(), "other"); !errors.Is(err, diagnosticscontract.ErrServiceStatusProvider) || calls.Load() != 2 {
		t.Fatal("invalid provider reached status source")
	}
}

func TestStatusCacheMarksFailureStaleBacksOffAndExpiresAtTenMinutes(t *testing.T) {
	base := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)
	var elapsed atomic.Int64
	var calls atomic.Int32
	var failed atomic.Bool
	service := NewServiceStatus(statusSourceFunc(func(context.Context, diagnosticscontract.ServiceStatusProvider) (diagnosticscontract.ServiceStatusSummary, error) {
		calls.Add(1)
		if failed.Load() {
			return diagnosticscontract.ServiceStatusSummary{}, errors.New("private upstream detail")
		}
		return fixtureStatusSummary(), nil
	}))
	service.now = func() time.Time { return base.Add(time.Duration(elapsed.Load())) }
	if _, err := service.Latest(context.Background(), diagnosticscontract.ServiceStatusOpenAI); err != nil {
		t.Fatal(err)
	}
	failed.Store(true)
	elapsed.Store(int64(time.Minute))
	stale, err := service.Latest(context.Background(), diagnosticscontract.ServiceStatusOpenAI)
	if err != nil || !stale.Stale || !stale.FetchedAt.Equal(base) || calls.Load() != 2 {
		t.Fatal("failed refresh was presented as fresh")
	}
	for i := 0; i < 10; i++ {
		if _, err := service.Latest(context.Background(), diagnosticscontract.ServiceStatusOpenAI); err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 2 {
		t.Fatal("failed source was not backed off")
	}
	elapsed.Store(int64(10 * time.Minute))
	if _, err := service.Latest(context.Background(), diagnosticscontract.ServiceStatusOpenAI); !errors.Is(err, diagnosticscontract.ErrServiceStatusUnavailable) {
		t.Fatal("ten-minute-old health snapshot was still served")
	}
	failed.Store(false)
	elapsed.Add(int64(statusRetryAfter))
	recovered, err := service.Latest(context.Background(), diagnosticscontract.ServiceStatusOpenAI)
	if err != nil || recovered.Stale || !recovered.FetchedAt.Equal(base.Add(time.Duration(elapsed.Load()))) {
		t.Fatal("source recovery did not update successful fetch time")
	}
	missing := NewServiceStatus(nil)
	for i := 0; i < 2; i++ {
		if _, err := missing.Latest(context.Background(), diagnosticscontract.ServiceStatusClaude); !errors.Is(err, diagnosticscontract.ErrServiceStatusUnavailable) {
			t.Fatal("initial source failure fabricated status data")
		}
	}
}

func TestStatusRefreshIsCoalescedAndCancelledWaiterCannotCancelOtherVisitors(t *testing.T) {
	var calls atomic.Int32
	started, release := make(chan struct{}), make(chan struct{})
	service := NewServiceStatus(statusSourceFunc(func(ctx context.Context, provider diagnosticscontract.ServiceStatusProvider) (diagnosticscontract.ServiceStatusSummary, error) {
		calls.Add(1)
		if provider == diagnosticscontract.ServiceStatusClaude {
			return fixtureStatusSummary(), nil
		}
		close(started)
		select {
		case <-release:
			return fixtureStatusSummary(), nil
		case <-ctx.Done():
			return diagnosticscontract.ServiceStatusSummary{}, ctx.Err()
		}
	}))
	ctx, cancel := context.WithCancel(context.Background())
	cancelled := make(chan error, 1)
	go func() { _, err := service.Latest(ctx, diagnosticscontract.ServiceStatusOpenAI); cancelled <- err }()
	<-started
	var group sync.WaitGroup
	results := make(chan error, 20)
	for i := 0; i < 20; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			_, err := service.Latest(context.Background(), diagnosticscontract.ServiceStatusOpenAI)
			results <- err
		}()
	}
	// A blocked OpenAI source must not serialize the unrelated Claude cache.
	if _, err := service.Latest(context.Background(), diagnosticscontract.ServiceStatusClaude); err != nil {
		t.Fatal(err)
	}
	cancel()
	if err := <-cancelled; !errors.Is(err, context.Canceled) {
		t.Fatal("caller cancellation was ignored")
	}
	close(release)
	group.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal("one caller cancelled the shared source refresh")
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("source requests were not coalesced per provider: %d", calls.Load())
	}
}

type statusHistorySourceFixture struct {
	statusSourceFunc
	history func(context.Context, diagnosticscontract.ServiceStatusProvider) (diagnosticscontract.ServiceStatusHistory, error)
}

func (f statusHistorySourceFixture) FetchHistory(ctx context.Context, provider diagnosticscontract.ServiceStatusProvider) (diagnosticscontract.ServiceStatusHistory, error) {
	return f.history(ctx, provider)
}

func fixtureStatusIncident(prefix string) diagnosticscontract.ServiceStatusIncident {
	base := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	stamp := func(offset time.Duration) *time.Time { value := base.Add(offset); return &value }
	return diagnosticscontract.ServiceStatusIncident{
		ID: prefix + "-incident", Name: prefix + " event", Status: "resolved", Impact: "minor", URL: "https://status.openai.com/incidents/fixture",
		CreatedAt: stamp(0), StartedAt: stamp(time.Minute), UpdatedAt: stamp(2 * time.Minute), ResolvedAt: stamp(3 * time.Minute),
		Components: []diagnosticscontract.ServiceStatusIncidentComponent{{ID: prefix + "-component", Name: prefix + " component", Status: "operational"}},
		IncidentUpdates: []diagnosticscontract.ServiceStatusIncidentUpdate{{
			ID: prefix + "-update", Status: "resolved", Body: prefix + " update", DisplayAt: stamp(4 * time.Minute), CreatedAt: stamp(5 * time.Minute), UpdatedAt: stamp(6 * time.Minute),
			AffectedComponents: []diagnosticscontract.ServiceStatusAffectedComponent{{Code: prefix + "-component", Name: prefix + " affected", OldStatus: "degraded_performance", NewStatus: "operational"}},
		}},
	}
}

func mutateStatusIncident(value *diagnosticscontract.ServiceStatusIncident) {
	value.Name, value.Components[0].Name, value.IncidentUpdates[0].Body = "changed event", "changed component", "changed update"
	value.IncidentUpdates[0].AffectedComponents[0].Name = "changed affected component"
	for _, stamp := range []*time.Time{value.CreatedAt, value.StartedAt, value.UpdatedAt, value.ResolvedAt, value.IncidentUpdates[0].DisplayAt, value.IncidentUpdates[0].CreatedAt, value.IncidentUpdates[0].UpdatedAt} {
		*stamp = time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	}
}

func assertStatusIncidentUnchanged(t *testing.T, value diagnosticscontract.ServiceStatusIncident, prefix string) {
	t.Helper()
	if len(value.Components) != 1 || len(value.IncidentUpdates) != 1 || len(value.IncidentUpdates[0].AffectedComponents) != 1 {
		t.Fatalf("incident nested records were lost: %#v", value)
	}
	if value.Name != prefix+" event" || value.Components[0].Name != prefix+" component" || value.IncidentUpdates[0].Body != prefix+" update" || value.IncidentUpdates[0].AffectedComponents[0].Name != prefix+" affected" {
		t.Fatalf("incident retained mutable nested slices: %#v", value)
	}
	base := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	for index, stamp := range []*time.Time{value.CreatedAt, value.StartedAt, value.UpdatedAt, value.ResolvedAt, value.IncidentUpdates[0].DisplayAt, value.IncidentUpdates[0].CreatedAt, value.IncidentUpdates[0].UpdatedAt} {
		if stamp == nil || !stamp.Equal(base.Add(time.Duration(index)*time.Minute)) {
			t.Fatalf("incident retained mutable timestamp %d: %v", index, stamp)
		}
	}
}

func TestStatusCacheDeepCopiesActiveAndHistoricalIncidentsAtBothBoundaries(t *testing.T) {
	summary := fixtureStatusSummary()
	summary.Incidents = []diagnosticscontract.ServiceStatusIncident{fixtureStatusIncident("active")}
	history := diagnosticscontract.ServiceStatusHistory{State: "ready", Coverage: "full", SourceURL: "https://untrusted.invalid/history", Incidents: []diagnosticscontract.ServiceStatusIncident{fixtureStatusIncident("history")}}
	var coreCalls, historyCalls atomic.Int32
	service := NewServiceStatus(statusHistorySourceFixture{
		statusSourceFunc: func(context.Context, diagnosticscontract.ServiceStatusProvider) (diagnosticscontract.ServiceStatusSummary, error) {
			coreCalls.Add(1)
			return summary, nil
		},
		history: func(context.Context, diagnosticscontract.ServiceStatusProvider) (diagnosticscontract.ServiceStatusHistory, error) {
			historyCalls.Add(1)
			return history, nil
		},
	})
	first, err := service.Latest(context.Background(), diagnosticscontract.ServiceStatusOpenAI)
	if err != nil || first.Stale || first.History.State != "ready" || first.History.Coverage != "partial" || first.History.SourceURL != "https://status.openai.com/history" || len(first.Summary.Incidents) != 1 || len(first.History.Incidents) != 1 {
		t.Fatalf("invalid incident snapshot: %#v %v", first, err)
	}
	// The cache and returned snapshot must not share the source's slices or dates.
	mutateStatusIncident(&summary.Incidents[0])
	mutateStatusIncident(&history.Incidents[0])
	assertStatusIncidentUnchanged(t, first.Summary.Incidents[0], "active")
	assertStatusIncidentUnchanged(t, first.History.Incidents[0], "history")
	// Neither may a caller mutate the copy stored for the next visitor.
	mutateStatusIncident(&first.Summary.Incidents[0])
	mutateStatusIncident(&first.History.Incidents[0])
	first.History.State, first.History.Coverage, first.History.SourceURL = "changed", "changed", "changed"
	second, err := service.Latest(context.Background(), diagnosticscontract.ServiceStatusOpenAI)
	if err != nil || coreCalls.Load() != 1 || historyCalls.Load() != 1 || !second.FetchedAt.Equal(first.FetchedAt) || second.History.State != "ready" || second.History.Coverage != "partial" || second.History.SourceURL != "https://status.openai.com/history" {
		t.Fatalf("cache hit lost history contract: %#v %v", second, err)
	}
	assertStatusIncidentUnchanged(t, second.Summary.Incidents[0], "active")
	assertStatusIncidentUnchanged(t, second.History.Incidents[0], "history")
}

func TestHistoryFailureDoesNotDiscardFreshCoreOrFabricateHistoricalIncidents(t *testing.T) {
	base := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)
	var coreCalls, historyCalls atomic.Int32
	var coreDeadline, historyDeadline, historyStarted time.Time
	service := NewServiceStatus(statusHistorySourceFixture{
		statusSourceFunc: func(ctx context.Context, _ diagnosticscontract.ServiceStatusProvider) (diagnosticscontract.ServiceStatusSummary, error) {
			coreCalls.Add(1)
			coreDeadline, _ = ctx.Deadline()
			value := fixtureStatusSummary()
			value.Incidents = []diagnosticscontract.ServiceStatusIncident{fixtureStatusIncident("active")}
			return value, nil
		},
		history: func(ctx context.Context, _ diagnosticscontract.ServiceStatusProvider) (diagnosticscontract.ServiceStatusHistory, error) {
			historyCalls.Add(1)
			historyStarted = time.Now()
			historyDeadline, _ = ctx.Deadline()
			return diagnosticscontract.ServiceStatusHistory{State: "ready", Incidents: []diagnosticscontract.ServiceStatusIncident{fixtureStatusIncident("discarded")}}, errors.New("private history failure")
		},
	})
	service.now = func() time.Time { return base }
	for i := 0; i < 2; i++ {
		value, err := service.Latest(context.Background(), diagnosticscontract.ServiceStatusClaude)
		if err != nil || value.Stale || !value.FetchedAt.Equal(base) || len(value.Summary.Components) != 1 || len(value.Summary.Incidents) != 1 || value.History.State != "unavailable" || value.History.Coverage != "partial" || value.History.SourceURL != "https://status.claude.com/history" || value.History.Incidents == nil || len(value.History.Incidents) != 0 {
			t.Fatalf("optional history failure damaged fresh core: %#v %v", value, err)
		}
	}
	if coreCalls.Load() != 1 || historyCalls.Load() != 1 {
		t.Fatal("optional history failure bypassed fresh core caching")
	}
	if coreDeadline.IsZero() || historyDeadline.IsZero() || historyDeadline.After(coreDeadline) || historyDeadline.After(historyStarted.Add(statusHistoryWait)) {
		t.Fatal("optional history exceeded its child or shared timeout budget")
	}
	withoutHistory := NewServiceStatus(statusSourceFunc(func(context.Context, diagnosticscontract.ServiceStatusProvider) (diagnosticscontract.ServiceStatusSummary, error) {
		return fixtureStatusSummary(), nil
	}))
	value, err := withoutHistory.Latest(context.Background(), diagnosticscontract.ServiceStatusOpenAI)
	if err != nil || value.Stale || value.History.State != "unavailable" || value.History.Incidents == nil || len(value.History.Incidents) != 0 || value.History.SourceURL != "https://status.openai.com/history" {
		t.Fatal("source without history capability invented historical results")
	}
}

func TestStatusHistoryBudgetReservesTimeForCurrentSummary(t *testing.T) {
	if budget := statusHistoryBudget(context.Background()); budget != statusHistoryWait {
		t.Fatalf("unbounded parent did not receive the history cap: %v", budget)
	}
	long, cancelLong := context.WithTimeout(context.Background(), time.Hour)
	defer cancelLong()
	if budget := statusHistoryBudget(long); budget != statusHistoryWait {
		t.Fatalf("long parent exceeded history cap: %v", budget)
	}
	short, cancelShort := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancelShort()
	if budget := statusHistoryBudget(short); budget <= 0 || budget > 50*time.Millisecond {
		t.Fatalf("short parent did not reserve half its remaining time: %v", budget)
	}
	expired, cancelExpired := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelExpired()
	if budget := statusHistoryBudget(expired); budget != 0 {
		t.Fatalf("expired parent started optional history: %v", budget)
	}
}

func TestCoreFailureMarksWholeIncidentSnapshotStaleAndExpiresItTogether(t *testing.T) {
	base := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)
	var elapsed atomic.Int64
	var failed atomic.Bool
	var historyCalls atomic.Int32
	service := NewServiceStatus(statusHistorySourceFixture{
		statusSourceFunc: func(context.Context, diagnosticscontract.ServiceStatusProvider) (diagnosticscontract.ServiceStatusSummary, error) {
			if failed.Load() {
				return diagnosticscontract.ServiceStatusSummary{}, errors.New("core unavailable")
			}
			value := fixtureStatusSummary()
			value.Incidents = []diagnosticscontract.ServiceStatusIncident{fixtureStatusIncident("active")}
			return value, nil
		},
		history: func(context.Context, diagnosticscontract.ServiceStatusProvider) (diagnosticscontract.ServiceStatusHistory, error) {
			historyCalls.Add(1)
			return diagnosticscontract.ServiceStatusHistory{State: "ready", Incidents: []diagnosticscontract.ServiceStatusIncident{fixtureStatusIncident("history")}}, nil
		},
	})
	service.now = func() time.Time { return base.Add(time.Duration(elapsed.Load())) }
	if _, err := service.Latest(context.Background(), diagnosticscontract.ServiceStatusOpenAI); err != nil {
		t.Fatal(err)
	}
	failed.Store(true)
	elapsed.Store(int64(time.Minute))
	value, err := service.Latest(context.Background(), diagnosticscontract.ServiceStatusOpenAI)
	if err != nil || !value.Stale || !value.FetchedAt.Equal(base) || value.History.State != "ready" || historyCalls.Load() != 1 {
		t.Fatalf("stale incident snapshot lost freshness provenance: %#v %v", value, err)
	}
	assertStatusIncidentUnchanged(t, value.Summary.Incidents[0], "active")
	assertStatusIncidentUnchanged(t, value.History.Incidents[0], "history")
	mutateStatusIncident(&value.Summary.Incidents[0])
	mutateStatusIncident(&value.History.Incidents[0])
	again, err := service.Latest(context.Background(), diagnosticscontract.ServiceStatusOpenAI)
	if err != nil || !again.Stale || !again.FetchedAt.Equal(base) {
		t.Fatal("stale cache changed successful fetch time")
	}
	assertStatusIncidentUnchanged(t, again.Summary.Incidents[0], "active")
	assertStatusIncidentUnchanged(t, again.History.Incidents[0], "history")
	elapsed.Store(int64(statusStaleFor))
	if _, err := service.Latest(context.Background(), diagnosticscontract.ServiceStatusOpenAI); !errors.Is(err, diagnosticscontract.ErrServiceStatusUnavailable) || historyCalls.Load() != 1 {
		t.Fatal("expired core/history snapshot was still served or history queried after core failure")
	}
}

type statusSeriesSourceFixture struct {
	statusSourceFunc
	series func(context.Context, diagnosticscontract.ServiceStatusProvider, diagnosticscontract.ServiceStatusSummary) ([]diagnosticscontract.ServiceStatusSeries, error)
}

func (f statusSeriesSourceFixture) FetchSeries(ctx context.Context, provider diagnosticscontract.ServiceStatusProvider, summary diagnosticscontract.ServiceStatusSummary) ([]diagnosticscontract.ServiceStatusSeries, error) {
	return f.series(ctx, provider, summary)
}

func TestStatusCacheDeepCopiesSeriesAndToleratesSeriesFailure(t *testing.T) {
	pct := 99.56
	sourceSeries := []diagnosticscontract.ServiceStatusSeries{{
		Key: "claude", Label: "Claude", ComponentCount: 1, Status: "operational", UptimePercent: &pct,
		Days: []diagnosticscontract.ServiceStatusSeriesDay{{Date: "2026-11-19", Status: "operational", Events: []diagnosticscontract.ServiceStatusSeriesEvent{{ID: "ev1", Name: "Original event"}}}},
	}}
	service := NewServiceStatus(statusSeriesSourceFixture{
		statusSourceFunc: func(context.Context, diagnosticscontract.ServiceStatusProvider) (diagnosticscontract.ServiceStatusSummary, error) {
			return fixtureStatusSummary(), nil
		},
		series: func(_ context.Context, provider diagnosticscontract.ServiceStatusProvider, _ diagnosticscontract.ServiceStatusSummary) ([]diagnosticscontract.ServiceStatusSeries, error) {
			if provider == diagnosticscontract.ServiceStatusOpenAI {
				return nil, errors.New("optional series failed")
			}
			return sourceSeries, nil
		},
	})
	first, err := service.Latest(context.Background(), diagnosticscontract.ServiceStatusClaude)
	if err != nil || len(first.Series) != 1 || first.Series[0].UptimePercent == nil || *first.Series[0].UptimePercent != 99.56 {
		t.Fatalf("series was not cached: %#v %v", first, err)
	}
	*sourceSeries[0].UptimePercent = 0
	sourceSeries[0].Days[0].Events[0].Name = "source mutated"
	*first.Series[0].UptimePercent = 12.34
	first.Series[0].Days[0].Events[0].Name = "caller mutated"
	second, err := service.Latest(context.Background(), diagnosticscontract.ServiceStatusClaude)
	if err != nil || *second.Series[0].UptimePercent != 99.56 || second.Series[0].Days[0].Events[0].Name != "Original event" {
		t.Fatalf("series cache retained mutable reference: %#v %v", second.Series, err)
	}
	fallback, err := service.Latest(context.Background(), diagnosticscontract.ServiceStatusOpenAI)
	if err != nil || len(fallback.Summary.Components) != 1 || len(fallback.Series) != 0 {
		t.Fatalf("optional series failure damaged core summary: %#v %v", fallback, err)
	}
}
