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
	level := new(slog.LevelVar)
	if raw := os.Getenv("PIXOMA_LOG_LEVEL"); raw != "" {
		var parsed slog.Level
		if err := parsed.UnmarshalText([]byte(raw)); err != nil {
			slog.Error("PIXOMA_LOG_LEVEL 无效", "err", err)
			os.Exit(1)
		}
		level.Set(parsed)
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

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
