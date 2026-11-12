package app

import (
	"context"
	"sync"
	"time"

	"github.com/dujiao-next/internal/logger"
)

type paidFulfillmentRecoverer interface{ RecoverPaidFulfillment() error }

// This resident loop runs in API and worker modes, including queue-disabled
// deployments. Database leases coordinate overlapping processes/restarts.
// Stop waits for the in-flight recovery. It does not change what that recovery writes.
type paidFulfillmentService struct {
	payments paidFulfillmentRecoverer
	interval time.Duration
	mu       sync.Mutex
	inflight int
	idle     chan struct{}
}

func (*paidFulfillmentService) Name() string { return "paid-fulfillment-outbox" }

func (s *paidFulfillmentService) Start(ctx context.Context) error {
	ticker := time.NewTicker(s.tick())
	defer ticker.Stop()
	for {
		if !s.enter(ctx) {
			return nil
		}
		if err := s.payments.RecoverPaidFulfillment(); err != nil {
			logger.Warnw("paid_fulfillment_recovery_failed", "error", err)
		}
		s.leave()
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (s *paidFulfillmentService) Stop(ctx context.Context) error {
	s.mu.Lock()
	if s.inflight == 0 {
		s.mu.Unlock()
		return nil
	}
	if s.idle == nil {
		s.idle = make(chan struct{})
	}
	idle := s.idle
	s.mu.Unlock()
	select {
	case <-idle:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *paidFulfillmentService) tick() time.Duration {
	if s.interval > 0 {
		return s.interval
	}
	return time.Minute
}

func (s *paidFulfillmentService) enter(ctx context.Context) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ctx.Err() != nil {
		return false
	}
	s.inflight++
	return true
}

func (s *paidFulfillmentService) leave() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.inflight > 0 {
		s.inflight--
	}
	if s.inflight == 0 && s.idle != nil {
		close(s.idle)
		s.idle = nil
	}
}
