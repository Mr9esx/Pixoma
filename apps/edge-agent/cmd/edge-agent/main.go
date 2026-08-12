package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mr9esx/comfyui_tgbot/apps/edge-agent/internal/pull"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob/factory"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/botconfig"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/actuator"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/comfyui"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		slog.Error("edge-agent failed", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	instID := envOr("INSTANCE_ID", "local")
	baseURL := envOr("CONTROL_PLANE_URL", envOr("PIXOMA_URL", "http://127.0.0.1:8080"))
	token := strings.TrimSpace(os.Getenv("AGENT_TOKEN"))
	if token == "" {
		return errString("AGENT_TOKEN is required")
	}
	comfyMock := envBool("COMFY_MOCK", true)
	comfyURL := envOr("COMFYUI_BASE_URL", "http://127.0.0.1:8188")
	wait := envDuration("CLAIM_WAIT", 25*time.Second)

	driver := envOr("BLOB_DRIVER", botconfig.BlobDriverLocalFS)
	blobStore, err := factory.New(driver, envOr("BLOB_LOCAL_ROOT", "data/blob"))
	if err != nil {
		return err
	}

	var comfy comfyui.Client
	if comfyMock {
		comfy = &comfyui.Mock{}
	} else {
		comfy, err = comfyui.NewClient(comfyui.Options{Mock: false, BaseURL: comfyURL})
		if err != nil {
			return err
		}
	}

	client := pull.NewClient(baseURL, token, instID)
	worker := &actuator.Worker{
		InstanceID: sharedkernel.InstanceID(instID),
		Comfy:      comfy,
		Blob:       blobStore,
		Status:     pull.NewStatusPublisher(client),
	}
	loop := &pull.Loop{Client: client, Worker: worker, Wait: wait}

	slog.Info("pixoma-edge-agent running",
		"instance_id", instID,
		"control_plane", baseURL,
		"mock", comfyMock,
		"blob_driver", driver,
	)
	return loop.Run(ctx)
}

type errString string

func (e errString) Error() string { return string(e) }

func envOr(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func envBool(k string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}

func envDuration(k string, def time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil || d < 0 {
		return def
	}
	return d
}
