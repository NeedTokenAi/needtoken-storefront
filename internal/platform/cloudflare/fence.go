package cloudflare

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
)

const (
	// DrainTimeout is the SIGTERM window used only while the Cloudflare profile
	// is on. Cloud Run keeps the historical 10 second default.
	DrainTimeout = 30 * time.Second
	FenceTTL     = 30 * time.Second
	FenceRefresh = 10 * time.Second
)

// ErrFenceHeld means another process already owns the single-writer fence.
var ErrFenceHeld = errors.New("cloudflare writer fence is held")

// FenceBackend is the compare-and-set surface used by the writer fence.
type FenceBackend interface {
	SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error)
	Get(ctx context.Context, key string) (value string, found bool, err error)
	CompareAndPExpire(ctx context.Context, key, value string, ttl time.Duration) (bool, error)
	CompareAndDelete(ctx context.Context, key, value string) (bool, error)
}

// FenceKey is the Redis key for the single writer. Prefix comes from the
// existing Redis prefix and is not a second database.
func FenceKey(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	return prefix + ":cf:writer"
}

// Claim acquires the fence or refreshes it when this process already owns it.
func Claim(ctx context.Context, backend FenceBackend, key, fence string, ttl time.Duration) error {
	if backend == nil {
		return errors.New("cloudflare writer fence backend is missing")
	}
	fence = strings.TrimSpace(fence)
	if fence == "" || strings.TrimSpace(key) == "" {
		return errors.New("cloudflare writer fence is missing")
	}
	ok, err := backend.SetNX(ctx, key, fence, ttl)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	current, found, err := backend.Get(ctx, key)
	if err != nil {
		return err
	}
	if !found || current != fence {
		return ErrFenceHeld
	}
	refreshed, err := backend.CompareAndPExpire(ctx, key, fence, ttl)
	if err != nil {
		return err
	}
	if !refreshed {
		return ErrFenceHeld
	}
	return nil
}

// Release drops the fence only when the value still belongs to this process.
func Release(ctx context.Context, backend FenceBackend, key, fence string) error {
	if backend == nil {
		return nil
	}
	_, err := backend.CompareAndDelete(ctx, key, strings.TrimSpace(fence))
	return err
}

// Keep refreshes the fence until ctx ends or a refresh loses ownership.
// The returned channel is closed only when ownership is lost.
func Keep(ctx context.Context, ticks <-chan time.Time, refresh func() error) <-chan struct{} {
	lost := make(chan struct{})
	go func() {
		failed := false
		defer func() {
			if failed {
				close(lost)
			}
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-ticks:
				if !ok {
					return
				}
				if err := refresh(); err != nil {
					failed = true
					return
				}
			}
		}
	}()
	return lost
}

// Hold claims the fence and refreshes it until release or ownership is lost.
func Hold(ctx context.Context, backend FenceBackend, key, fence string, ttl, refreshEvery time.Duration) (func(), <-chan struct{}, error) {
	if err := Claim(ctx, backend, key, fence, ttl); err != nil {
		return nil, nil, err
	}
	if refreshEvery <= 0 {
		refreshEvery = FenceRefresh
	}
	holdCtx, cancel := context.WithCancel(ctx)
	ticks := time.NewTicker(refreshEvery)
	lost := Keep(holdCtx, ticks.C, func() error {
		ok, err := backend.CompareAndPExpire(holdCtx, key, strings.TrimSpace(fence), ttl)
		if err != nil {
			return err
		}
		if !ok {
			return ErrFenceHeld
		}
		return nil
	})
	var once sync.Once
	release := func() {
		once.Do(func() {
			cancel()
			ticks.Stop()
			_ = Release(context.Background(), backend, key, fence)
		})
	}
	return release, lost, nil
}
