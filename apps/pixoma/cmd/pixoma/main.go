package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Mr9esx/Pixoma/apps/pixoma/internal/application"
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
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var liveDemo bool
	flag.BoolVar(&liveDemo, "livedemo", false, "start an isolated readonly live demo")
	flag.Parse()

	opts := application.OptionsFromEnv()
	var err error
	if liveDemo {
		err = application.RunLiveDemo(ctx, opts)
	} else {
		err = application.Run(ctx, opts)
	}
	if err != nil {
		slog.Error("pixoma failed", "err", err)
		os.Exit(1)
	}
}
