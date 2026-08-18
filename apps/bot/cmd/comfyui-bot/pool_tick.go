package main

import (
	"context"
	"log/slog"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
)

// tickPool reloads the instance list from DB so admin-api writes become
// visible to bot scheduling within one probe cycle.
func tickPool(ctx context.Context, pool *edge.Pool) {
	if err := pool.Refresh(ctx); err != nil {
		slog.Warn("instance pool refresh", "err", err)
	}
}
