package presence

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Mr9esx/Pixoma/apps/edge-agent/internal/controlplane"
	edge "github.com/Mr9esx/Pixoma/internal/edge/domain"
	"github.com/Mr9esx/Pixoma/internal/tasks/infrastructure/comfyui"
)

const defaultEvery = 5 * time.Second
const probeTimeout = 2 * time.Second
const defaultMetricsInterval = 30 * time.Second
const minMetricsInterval = 5 * time.Second

// Reporter probes local Comfy and POSTs /agent/v1/presence.
type Reporter struct {
	Client          *controlplane.Client
	Comfy           comfyui.Client
	Collect         func(ctx context.Context) edge.Hardware
	Sample          func(ctx context.Context, since time.Time) edge.Metrics
	Every           time.Duration
	MetricsInterval time.Duration
	SendHardware    bool
	refreshNext     bool
	lastMetrics     time.Time
	startedAt       time.Time
	lastConsuming   *bool
}

func (r *Reporter) interval() time.Duration {
	if r != nil && r.Every > 0 {
		return r.Every
	}
	return defaultEvery
}

func (r *Reporter) metricsInterval() time.Duration {
	if r != nil && r.MetricsInterval >= minMetricsInterval {
		return r.MetricsInterval
	}
	return defaultMetricsInterval
}

func (r *Reporter) ProbeAndReport(ctx context.Context) error {
	if r == nil || r.Client == nil {
		return fmt.Errorf("presence: client not configured")
	}
	if r.startedAt.IsZero() {
		r.startedAt = time.Now().UTC()
	}
	running := false
	comfyVersion := ""
	if r.Comfy != nil {
		probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
		st, err := r.Comfy.SystemStats(probeCtx)
		cancel()
		running = err == nil && st != nil && st.Reachable
		if st != nil {
			comfyVersion = st.ComfyUIVersion
		}
	}
	var hw *edge.Hardware
	if (r.SendHardware || r.refreshNext) && r.Collect != nil {
		collected := r.Collect(ctx)
		hw = &collected
	}
	var m *edge.Metrics
	now := time.Now().UTC()
	if r.Sample != nil && (r.lastMetrics.IsZero() || now.Sub(r.lastMetrics) >= r.metricsInterval()) {
		collected := r.Sample(ctx, r.lastMetrics)
		r.lastMetrics = now
		m = &collected
	}
	refresh, consuming, err := r.Client.ReportPresence(ctx, running, r.startedAt, comfyVersion, hw, m)
	if err != nil {
		return err
	}
	if !consuming && (r.lastConsuming == nil || *r.lastConsuming) {
		slog.Warn("edge has no dispatch topic binding; it will not receive tasks",
			"edge_id", r.Client.EdgeID,
			"hint", "bind a topic in the admin (PATCH /api/v1/edges/{id})",
		)
	}
	r.lastConsuming = &consuming
	r.SendHardware = false
	r.refreshNext = refresh
	return nil
}

func (r *Reporter) Run(ctx context.Context) error {
	if err := r.ProbeAndReport(ctx); err != nil {
		slog.Warn("presence report failed", "err", err)
	}
	ticker := time.NewTicker(r.interval())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := r.ProbeAndReport(ctx); err != nil {
				slog.Warn("presence report failed", "err", err)
			}
		}
	}
}
