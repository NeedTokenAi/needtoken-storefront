package app

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingService struct {
	started bool
}

func (s *recordingService) Name() string { return "recording" }
func (s *recordingService) Start(context.Context) error {
	s.started = true
	return nil
}
func (s *recordingService) Stop(context.Context) error { return nil }

func TestRunnerDoesNotStartWhenContextIsCanceled(t *testing.T) {
	svc := &recordingService{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := NewRunner(svc).Run(ctx, time.Second, zap.NewNop().Sugar())
	require.ErrorContains(t, err, "background start refused")
	require.False(t, svc.started)
}
