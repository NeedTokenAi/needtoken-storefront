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
