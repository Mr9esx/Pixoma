package application

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/Mr9esx/Pixoma/internal/channels/domain"
)

const defaultProbeInterval = 30 * time.Second

// ReachabilityProbe periodically probes enabled channels and persists last_check.
// GET and page render must not wait on it; POST /probe may kick ProbeOnce in the background.
type ReachabilityProbe struct {
	Svc      *Service
	Interval time.Duration
	mu       sync.Mutex
}

func (p *ReachabilityProbe) interval() time.Duration {
	if p != nil && p.Interval > 0 {
		return p.Interval
	}
	return defaultProbeInterval
}

// ProbeOnce checks every enabled channel and writes last_check. One channel
// failure does not stop the rest. List failure is returned.
func (p *ReachabilityProbe) ProbeOnce(ctx context.Context) error {
	if p == nil || p.Svc == nil {
		return nil
	}
	if !p.mu.TryLock() {
		return nil
	}
	defer p.mu.Unlock()
	chs, err := p.Svc.List(ctx)
	if err != nil {
		return err
	}
	for _, ch := range chs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !ch.Enabled || ch.Platform == string(domain.PlatformMCP) {
			continue
		}
		res, err := p.Svc.CheckReachability(ctx, ch.ID)
		if err != nil {
			slog.Warn("channel reachability probe", "channel", ch.ID, "err", err)
			continue
		}
		slog.Info("channel reachability probe", "channel", ch.ID, "kind", string(res.Kind))
	}
	return nil
}

// Run probes immediately, then on each interval until ctx is cancelled.
func (p *ReachabilityProbe) Run(ctx context.Context) error {
	if err := p.ProbeOnce(ctx); err != nil && ctx.Err() == nil {
		slog.Error("channel reachability probe", "err", err)
	}
	ticker := time.NewTicker(p.interval())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := p.ProbeOnce(ctx); err != nil && ctx.Err() == nil {
				slog.Error("channel reachability probe", "err", err)
			}
		}
	}
}
