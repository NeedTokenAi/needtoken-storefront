package cloudflare

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const IsolationIsolated = "isolated"

// ErrIsolationRequired means the process was not started as an isolated
// candidate. Holding this permit does not stop the existing Cloud Run writer.
var ErrIsolationRequired = errors.New("cloudflare profile: candidate isolation must be isolated; this lease does not stop the existing Cloud Run writer")

// WriterPermit is the Redis lease for this new process only.
// The existing Cloud Run image does not read it.
type WriterPermit struct {
	ctx     context.Context
	cancel  context.CancelFunc
	release func()
	lost    <-chan struct{}
}

// OpenWriterPermit claims the lease after the isolation check and before any
// database, migration, default-admin, or background work.
func OpenWriterPermit(parent context.Context, backend FenceBackend, key, fence, isolation string) (*WriterPermit, error) {
	if strings.TrimSpace(isolation) != IsolationIsolated {
		return nil, ErrIsolationRequired
	}
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	release, lost, err := Hold(ctx, backend, key, fence, FenceTTL, FenceRefresh)
	if err != nil {
		cancel()
		return nil, err
	}
	return &WriterPermit{ctx: ctx, cancel: cancel, release: release, lost: lost}, nil
}

// Context stops when the lease is lost or the process begins shutdown.
func (p *WriterPermit) Context() context.Context {
	if p == nil {
		return context.Background()
	}
	return p.ctx
}

// Lost reports that this process no longer holds the lease.
func (p *WriterPermit) Lost() <-chan struct{} {
	if p == nil {
		return nil
	}
	return p.lost
}

// Cancel asks running services to stop. It does not release the lease.
func (p *WriterPermit) Cancel() {
	if p == nil || p.cancel == nil {
		return
	}
	p.cancel()
}

// Allow is the check before database, migration, default admin, and background start.
func (p *WriterPermit) Allow(step string) error {
	if p == nil {
		return errors.New("cloudflare profile: writer permit is missing")
	}
	select {
	case <-p.lost:
		return fmt.Errorf("cloudflare profile: writer permit lost before %s; this lease does not stop the existing Cloud Run writer", step)
	default:
		return nil
	}
}

// DrainThenRelease stops background work and waits for drain, then drops the lease.
// HTTP 503 alone is not this step.
func (p *WriterPermit) DrainThenRelease(drain func(context.Context) error) error {
	if p == nil {
		return nil
	}
	p.Cancel()
	var err error
	if drain != nil {
		ctx, cancel := context.WithTimeout(context.Background(), DrainTimeout)
		err = drain(ctx)
		cancel()
	}
	if p.release != nil {
		p.release()
	}
	return err
}

// Release drops the lease. Call it only after background work has drained.
func (p *WriterPermit) Release() {
	if p == nil || p.release == nil {
		return
	}
	p.release()
}
