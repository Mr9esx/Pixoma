package application

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Mr9esx/Pixoma/internal/platform/botconfig"
)

type Config struct {
	EdgeID          string
	ControlPlaneURL string
	AgentToken      string
	ComfyURL        string
	ClaimWait       time.Duration
	MetricsInterval time.Duration
	BlobDriver      string
	BlobLocalRoot   string
}

func FromEnv() (Config, error) {
	cfg := Config{
		EdgeID:          envOr("EDGE_ID", "local"),
		ControlPlaneURL: envOr("CONTROL_PLANE_URL", envOr("PIXOMA_URL", "http://127.0.0.1:8082")),
		AgentToken:      strings.TrimSpace(os.Getenv("AGENT_TOKEN")),
		ComfyURL:        envOr("COMFYUI_BASE_URL", "http://127.0.0.1:8188"),
		ClaimWait:       envDuration("CLAIM_WAIT", 5*time.Second),
		MetricsInterval: envDuration("METRICS_INTERVAL", 30*time.Second),
		BlobDriver:      envOr("BLOB_DRIVER", botconfig.BlobDriverLocalFS),
		BlobLocalRoot:   envOr("BLOB_LOCAL_ROOT", "data/blob"),
	}
	if cfg.AgentToken == "" {
		return Config{}, fmt.Errorf("AGENT_TOKEN is required")
	}
	return cfg, nil
}

func envOr(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
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
