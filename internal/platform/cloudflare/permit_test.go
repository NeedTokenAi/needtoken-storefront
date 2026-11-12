package cloudflare

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenWriterPermitDoesNotClaimWithoutIsolation(t *testing.T) {
	backend := &memoryFence{}
	_, err := OpenWriterPermit(context.Background(), backend, "dj:cf:writer", "fence-1", "production")
	require.ErrorIs(t, err, ErrIsolationRequired)
	require.ErrorContains(t, err, "does not stop the existing Cloud Run writer")
	require.False(t, backend.held)
}

func TestLostPermitBlocksStartupAndDrainPrecedesRelease(t *testing.T) {
	lost := make(chan struct{})
	close(lost)
	var order []string
	permit := &WriterPermit{
		ctx:     context.Background(),
		cancel:  func() { order = append(order, "cancel") },
		release: func() { order = append(order, "release") },
		lost:    lost,
	}
	for _, step := range []string{"database", "migrations", "default-admin", "background"} {
		err := permit.Allow(step)
		require.ErrorContains(t, err, step)
		require.ErrorContains(t, err, "does not stop the existing Cloud Run writer")
	}
	require.NoError(t, permit.DrainThenRelease(func(context.Context) error {
		order = append(order, "drain")
		return nil
	}))
	require.Equal(t, []string{"cancel", "drain", "release"}, order)
}

func TestIsolatedPermitIsClaimedBeforeDatabase(t *testing.T) {
	backend := &memoryFence{}
	permit, err := OpenWriterPermit(context.Background(), backend, "dj:cf:writer", "fence-1", IsolationIsolated)
	require.NoError(t, err)
	require.True(t, backend.held)
	require.NoError(t, permit.Allow("database"))
	permit.Release()
	require.False(t, backend.held)
}
