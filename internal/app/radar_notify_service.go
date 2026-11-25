package app

import (
	"context"
	"time"

	"github.com/dujiao-next/internal/logger"
)

type radarResetPoller interface {
	PollAndNotify(ctx context.Context) error
}

// radarNotifyService periodically polls public reset feeds and emails subscribed
// members when new reset announcements appear. Runs in-process so API-only
// Cloudflare Container deployments work without an external queue worker.
type radarNotifyService struct {
	poller   radarResetPoller
	interval time.Duration
}

func (*radarNotifyService) Name() string { return "radar-reset-notifier" }

func (s *radarNotifyService) Start(ctx context.Context) error {
	if s == nil || s.poller == nil {
		return nil
	}
	interval := s.interval
	if interval <= 0 {
		interval = 5 * time.Minute
	}

	// Delay the initial baseline check briefly so HTTP startup finishes cleanly.
	startupTimer := time.NewTimer(15 * time.Second)
	select {
	case <-ctx.Done():
		startupTimer.Stop()
		return nil
	case <-startupTimer.C:
	}

	if err := s.poller.PollAndNotify(ctx); err != nil && ctx.Err() == nil {
		logger.Warnw("radar_reset_poll_failed", "error", err)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := s.poller.PollAndNotify(ctx); err != nil && ctx.Err() == nil {
				logger.Warnw("radar_reset_poll_failed", "error", err)
			}
		}
	}
}

func (*radarNotifyService) Stop(context.Context) error { return nil }
