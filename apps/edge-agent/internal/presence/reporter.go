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
	lastRunning     *bool
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
	probeError := "未配置 ComfyUI 客户端"
	if r.Comfy != nil {
		probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
		st, err := r.Comfy.SystemStats(probeCtx)
		cancel()
		running = err == nil && st != nil && st.Reachable
		if err != nil {
			probeError = err.Error()
		}
		if st != nil {
			comfyVersion = st.ComfyUIVersion
			if st.Error != "" {
				probeError = st.Error
			}
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
	if r.lastRunning == nil || *r.lastRunning != running {
		if running {
			slog.Info("ComfyUI 已连接", "edge_id", r.Client.EdgeID, "version", comfyVersion)
		} else {
			slog.Warn("ComfyUI 无法连接", "edge_id", r.Client.EdgeID, "reason", probeError)
		}
	}
	r.lastRunning = &running
	if !consuming && (r.lastConsuming == nil || *r.lastConsuming) {
		slog.Warn("Edge Agent 没有任务主题绑定",
			"edge_id", r.Client.EdgeID,
			"hint", "在管理页面绑定任务主题（PATCH /api/v1/edges/{id}）",
		)
	}
	if consuming && r.lastConsuming != nil && !*r.lastConsuming {
		slog.Info("Edge Agent 任务主题绑定已恢复", "edge_id", r.Client.EdgeID)
	}
	r.lastConsuming = &consuming
	r.SendHardware = false
	r.refreshNext = refresh
	return nil
}

func (r *Reporter) Run(ctx context.Context) error {
	if err := r.ProbeAndReport(ctx); err != nil {
		slog.Warn("Edge Agent 上报状态失败", "edge_id", r.Client.EdgeID, "err", err)
	}
	ticker := time.NewTicker(r.interval())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := r.ProbeAndReport(ctx); err != nil {
				slog.Warn("Edge Agent 上报状态失败", "edge_id", r.Client.EdgeID, "err", err)
			}
		}
	}
}
