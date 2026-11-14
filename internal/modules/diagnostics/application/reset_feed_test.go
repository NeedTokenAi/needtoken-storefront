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

type feedSourceFunc func(context.Context) (diagnosticscontract.ResetFeedPage, error)

func (f feedSourceFunc) Fetch(ctx context.Context) (diagnosticscontract.ResetFeedPage, error) {
	return f(ctx)
}

func fixtureFeedPage() diagnosticscontract.ResetFeedPage {
	return diagnosticscontract.ResetFeedPage{HasMore: true, Records: []diagnosticscontract.ResetRecord{{ID: "123", Type: "regular", Text: "original", Source: diagnosticscontract.ResetSource{Type: "observed"}}}}
}

func TestResetFeedCachesSuccessAndBoundsStaleFallbackWithFailureBackoff(t *testing.T) {
	base := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)
	var elapsed atomic.Int64
	var failed atomic.Bool
	var calls atomic.Int32
	service := NewResetFeed(feedSourceFunc(func(context.Context) (diagnosticscontract.ResetFeedPage, error) {
		calls.Add(1)
		if failed.Load() {
			return diagnosticscontract.ResetFeedPage{}, errors.New("private upstream detail")
		}
		return fixtureFeedPage(), nil
	}))
	service.now = func() time.Time { return base.Add(time.Duration(elapsed.Load())) }
	first, err := service.Latest(context.Background())
	if err != nil || first.Stale || first.SourceURL != resetFeedSourceURL || !first.FetchedAt.Equal(base) || !first.HasMore {
		t.Fatalf("invalid first snapshot: %#v %v", first, err)
	}
	first.Records[0].Text = "caller changed"
	elapsed.Store(int64(59 * time.Second))
	cached, err := service.Latest(context.Background())
	if err != nil || calls.Load() != 1 || cached.Records[0].Text != "original" || !cached.FetchedAt.Equal(base) {
		t.Fatal("cache hit changed data or successful fetch time")
	}
	failed.Store(true)
	elapsed.Store(int64(61 * time.Second))
	stale, err := service.Latest(context.Background())
	if err != nil || !stale.Stale || !stale.FetchedAt.Equal(base) || calls.Load() != 2 {
		t.Fatalf("failed refresh lost stale snapshot: %#v %v", stale, err)
	}
	for i := 0; i < 20; i++ {
		if _, err := service.Latest(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 2 {
		t.Fatal("failed refresh was not backed off")
	}
	elapsed.Store(int64(12*time.Hour + time.Second))
	if _, err := service.Latest(context.Background()); !errors.Is(err, diagnosticscontract.ErrResetFeedUnavailable) {
		t.Fatal("expired snapshot served past 12 hours")
	}
	failed.Store(false)
	elapsed.Add(int64(resetRetryAfter))
	recovered, err := service.Latest(context.Background())
	if err != nil || recovered.Stale || !recovered.FetchedAt.Equal(base.Add(time.Duration(elapsed.Load()))) {
		t.Fatalf("refresh did not recover: %#v %v", recovered, err)
	}
}

func TestResetFeedBacksOffInitialFailureAndAcceptsValidEmptyPage(t *testing.T) {
	var calls atomic.Int32
	service := NewResetFeed(feedSourceFunc(func(context.Context) (diagnosticscontract.ResetFeedPage, error) {
		calls.Add(1)
		return diagnosticscontract.ResetFeedPage{}, diagnosticscontract.ErrResetFeedUnavailable
	}))
	for i := 0; i < 20; i++ {
		if _, err := service.Latest(context.Background()); !errors.Is(err, diagnosticscontract.ErrResetFeedUnavailable) {
			t.Fatal("initial failure was not unavailable")
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("initial failures hit upstream repeatedly: %d", calls.Load())
	}
	empty := NewResetFeed(feedSourceFunc(func(context.Context) (diagnosticscontract.ResetFeedPage, error) {
		return diagnosticscontract.ResetFeedPage{}, nil
	}))
	snapshot, err := empty.Latest(context.Background())
	if err != nil || snapshot.Stale || snapshot.Records == nil || len(snapshot.Records) != 0 {
		t.Fatalf("valid empty snapshot rejected: %#v %v", snapshot, err)
	}
}

func TestResetFeedCoalescesRefreshAndCancellationOnlyStopsItsWaiter(t *testing.T) {
	var calls atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	service := NewResetFeed(feedSourceFunc(func(ctx context.Context) (diagnosticscontract.ResetFeedPage, error) {
		calls.Add(1)
		close(started)
		select {
		case <-release:
			return fixtureFeedPage(), nil
		case <-ctx.Done():
			return diagnosticscontract.ResetFeedPage{}, ctx.Err()
		}
	}))
	ctx, cancel := context.WithCancel(context.Background())
	cancelled := make(chan error, 1)
	go func() { _, err := service.Latest(ctx); cancelled <- err }()
	<-started
	var group sync.WaitGroup
	results := make(chan error, 20)
	for i := 0; i < 20; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			value, err := service.Latest(context.Background())
			if err == nil && (len(value.Records) != 1 || value.Stale) {
				err = errors.New("invalid merged snapshot")
			}
			results <- err
		}()
	}
	cancel()
	if err := <-cancelled; !errors.Is(err, context.Canceled) {
		t.Fatalf("waiter ignored cancellation: %v", err)
	}
	close(release)
	group.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("refresh not shared: %d", calls.Load())
	}
	before := calls.Load()
	if _, err := service.Latest(ctx); !errors.Is(err, context.Canceled) || calls.Load() != before {
		t.Fatal("already-cancelled request reached source")
	}
}
