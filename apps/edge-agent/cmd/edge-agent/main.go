package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Mr9esx/Pixoma/apps/edge-agent/internal/application"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := application.FromEnv()
	if err != nil {
		slog.Error("edge-agent failed", "err", err)
		os.Exit(1)
	}
	if err := application.Run(ctx, cfg); err != nil {
		slog.Error("edge-agent failed", "err", err)
		os.Exit(1)
	}
}
