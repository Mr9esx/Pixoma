package main

import (
	"context"
	"log/slog"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance"
)

// tickPool reloads the instance list from DB then probes health.
// Used on each health_probe_interval tick so admin-api writes become
// visible to bot scheduling within one probe cycle.
func tickPool(ctx context.Context, pool *instance.Pool) {
	if err := pool.Refresh(ctx); err != nil {
		slog.Warn("instance pool refresh", "err", err)
	}
	pool.Probe(ctx)
}
