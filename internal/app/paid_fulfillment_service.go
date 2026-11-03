package app

import (
	"context"
	"github.com/dujiao-next/internal/logger"
	"time"
)

type paidFulfillmentRecoverer interface{ RecoverPaidFulfillment() error }

// This resident loop runs in API and worker modes, including queue-disabled
// deployments. Database leases coordinate overlapping processes/restarts.
type paidFulfillmentService struct{ payments paidFulfillmentRecoverer }

func (*paidFulfillmentService) Name() string { return "paid-fulfillment-outbox" }
func (s *paidFulfillmentService) Start(ctx context.Context) error {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		if err := s.payments.RecoverPaidFulfillment(); err != nil {
			logger.Warnw("paid_fulfillment_recovery_failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
func (*paidFulfillmentService) Stop(context.Context) error { return nil }
