package jobs

import (
	"context"
	"testing"

	"github.com/dujiao-next/internal/config"
	"github.com/stretchr/testify/require"
)

func TestDisabledConsumptionDoesNotOpenTheQueue(t *testing.T) {
	svc, err := NewService(&config.QueueConfig{Enabled: true, Host: "production.upstash.example"}, nil, false)
	require.NoError(t, err)
	require.Nil(t, svc.server)
	require.Nil(t, svc.scheduler)
	require.True(t, svc.idle)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.NoError(t, svc.Start(ctx))
	require.NoError(t, svc.Stop(context.Background()))
}
