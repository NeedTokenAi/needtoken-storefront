package app

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type recoverFunc func() error

func (f recoverFunc) RecoverPaidFulfillment() error { return f() }

func TestPaidFulfillmentStopWaitsAndDoesNotStartAnotherRound(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	svc := &paidFulfillmentService{
		interval: 5 * time.Millisecond,
		payments: recoverFunc(func() error {
			if calls.Add(1) == 1 {
				close(started)
				<-release
			}
			return nil
		}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = svc.Start(ctx)
		close(done)
	}()
	<-started
	cancel()
	stopped := make(chan error, 1)
	go func() {
		stopped <- svc.Stop(context.Background())
	}()
	select {
	case <-stopped:
		t.Fatal("stop returned before the in-flight recovery finished")
	case <-time.After(30 * time.Millisecond):
	}
	close(release)
	require.NoError(t, <-stopped)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("service did not leave after cancel")
	}
	require.Equal(t, int32(1), calls.Load())
}
