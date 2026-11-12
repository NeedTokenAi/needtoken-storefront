package cloudflare

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type memoryFence struct {
	mu    sync.Mutex
	value string
	held  bool
}

func (m *memoryFence) SetNX(_ context.Context, _, value string, _ time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.held {
		return false, nil
	}
	m.value = value
	m.held = true
	return true, nil
}

func (m *memoryFence) Get(context.Context, string) (string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.held {
		return "", false, nil
	}
	return m.value, true, nil
}

func (m *memoryFence) CompareAndPExpire(_ context.Context, _, value string, _ time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.held || m.value != value {
		return false, nil
	}
	return true, nil
}

func (m *memoryFence) CompareAndDelete(_ context.Context, _, value string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.held || m.value != value {
		return false, nil
	}
	m.held = false
	m.value = ""
	return true, nil
}

func (m *memoryFence) force(value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.held = true
	m.value = value
}

func TestFenceIsSingleWriter(t *testing.T) {
	backend := &memoryFence{}
	ctx := context.Background()
	require.NoError(t, Claim(ctx, backend, "dj:cf:writer", "1", time.Second))
	require.ErrorIs(t, Claim(ctx, backend, "dj:cf:writer", "2", time.Second), ErrFenceHeld)
	require.NoError(t, Claim(ctx, backend, "dj:cf:writer", "1", time.Second))
	require.NoError(t, Release(ctx, backend, "dj:cf:writer", "2"))
	current, found, err := backend.Get(ctx, "dj:cf:writer")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "1", current)
	require.NoError(t, Release(ctx, backend, "dj:cf:writer", "1"))
	_, found, err = backend.Get(ctx, "dj:cf:writer")
	require.NoError(t, err)
	require.False(t, found)
	require.NoError(t, Claim(ctx, backend, "dj:cf:writer", "2", time.Second))
}

func TestKeepStopsWhenFenceIsLost(t *testing.T) {
	backend := &memoryFence{}
	require.NoError(t, Claim(context.Background(), backend, "dj:cf:writer", "1", time.Second))
	ticks := make(chan time.Time, 1)
	lost := Keep(context.Background(), ticks, func() error {
		ok, err := backend.CompareAndPExpire(context.Background(), "dj:cf:writer", "1", time.Second)
		if err != nil {
			return err
		}
		if !ok {
			return ErrFenceHeld
		}
		return nil
	})
	backend.force("9")
	ticks <- time.Now()
	select {
	case <-lost:
	case <-time.After(time.Second):
		t.Fatal("lost fence was not reported")
	}
}

func TestFenceKeyUsesRedisPrefix(t *testing.T) {
	require.Equal(t, "dj:cf:writer", FenceKey("dj"))
	require.Equal(t, "store:cf:writer", FenceKey(" store "))
}
