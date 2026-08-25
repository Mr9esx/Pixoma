package botconfig

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	RuntimeModeAllinone = "allinone" // deprecated alias of local
	RuntimeModeSplit    = "split"    // deprecated alias of remote

	PlacementLocal  = "local"
	PlacementRemote = "remote"

	QueueDriverMemory = "memory"
	QueueDriverRedis  = "redis"

	BlobDriverLocalFS = "localfs"
	BlobDriverSharedFS = "sharedfs"
	BlobDriverS3      = "s3"
	BlobDriverTOS     = "tos"
)

// Config is the bot process configuration.
type Config struct {
	HTTPAddr       string `yaml:"http_addr"`
	DatabaseDSN    string `yaml:"database_dsn"`
	BlobRoot       string `yaml:"blob_root"`
	ComfyUIBaseURL string `yaml:"comfyui_base_url"`
	// Edges optionally seeds multiple ComfyUI instances on startup.
	// When empty, no default instance is upserted; nodes are added manually.
	Edges []EdgeSeed `yaml:"edges"`
	// ComfyMock enables the in-process ComfyUI mock (default true).
	// Set false (or COMFY_MOCK=0) to call a real ComfyUI at ComfyUIBaseURL.
	ComfyMock bool `yaml:"comfy_mock"`
	// HealthProbeInterval is how often enabled real Comfy instances are probed
	// via SystemStats (default 30s). Parsed as Go duration, e.g. "30s".
	HealthProbeInterval string `yaml:"health_probe_interval"`

	// Placement is local (same machine) or remote (Edge pulls over the network).
	// RuntimeMode is a deprecated alias (allinone=local, split=remote).
	Placement   string      `yaml:"placement"`
	RuntimeMode string      `yaml:"runtime_mode"`
	Queue       QueueConfig `yaml:"queue"`
	Blob        BlobConfig  `yaml:"blob"`
}

// QueueConfig selects the queue adapter.
type QueueConfig struct {
	Driver string `yaml:"driver"` // memory | redis
}

// BlobConfig selects the blob adapter (BlobRoot still used for localfs).
type BlobConfig struct {
	Driver string `yaml:"driver"` // localfs | s3 | tos
	// TOS holds non-secret connection fields; keys stay in env only.
	TOS BlobTOSConfig `yaml:"tos"`
	// S3 holds non-secret connection fields; keys stay in env only.
	S3 BlobTOSConfig `yaml:"s3"`
}

// BlobTOSConfig is optional YAML for TOS endpoint/region/bucket.
type BlobTOSConfig struct {
	Endpoint string `yaml:"endpoint"`
	Region   string `yaml:"region"`
	Bucket   string `yaml:"bucket"`
}

// EdgeSeed is one row under comfy_instances in bot YAML.
type EdgeSeed struct {
	ID           string   `yaml:"id"`
	BaseURL      string   `yaml:"base_url"`
	Enabled      *bool    `yaml:"enabled"`
	Capabilities []string `yaml:"capabilities"`
}

func Default() Config {
	return Config{
		HTTPAddr:            ":8082",
		BlobRoot:            "data/blob",
		ComfyUIBaseURL:      "http://127.0.0.1:8188",
		ComfyMock:           true,
		HealthProbeInterval: "30s",
		Placement:           PlacementLocal,
		RuntimeMode:         RuntimeModeAllinone,
		Queue:               QueueConfig{Driver: QueueDriverMemory},
		Blob:                BlobConfig{Driver: BlobDriverLocalFS},
	}
}

// ValidateRuntimeDrivers checks local/remote vs blob. Queue driver is not required.
func (c Config) ValidateRuntimeDrivers() error {
	place := c.resolvedPlacement()
	b := strings.TrimSpace(c.Blob.Driver)
	if b == "" {
		b = BlobDriverLocalFS
	}
	switch place {
	case PlacementLocal:
		switch b {
		case BlobDriverLocalFS, BlobDriverS3, BlobDriverTOS:
			return nil
		default:
			return fmt.Errorf("botconfig: unknown blob.driver %q", b)
		}
	case PlacementRemote:
		if b == BlobDriverLocalFS {
			return fmt.Errorf("botconfig: remote requires blob.driver=s3 or tos, got %q", b)
		}
		if b != BlobDriverS3 && b != BlobDriverTOS {
			return fmt.Errorf("botconfig: remote requires blob.driver=s3 or tos, got %q", b)
		}
		return nil
	default:
		return fmt.Errorf("botconfig: unknown placement %q", place)
	}
}

func (c Config) resolvedPlacement() string {
	p := strings.TrimSpace(c.Placement)
	mode := strings.TrimSpace(c.RuntimeMode)
	if p == PlacementRemote || p == RuntimeModeSplit || mode == PlacementRemote || mode == RuntimeModeSplit {
		return PlacementRemote
	}
	if p != "" && p != PlacementLocal && p != RuntimeModeAllinone {
		return p
	}
	if mode != "" && mode != RuntimeModeAllinone && mode != PlacementLocal {
		return mode
	}
	return PlacementLocal
}

// Load reads optional YAML then applies env overrides.
// Empty path tries BOT_CONFIG, then configs/bot.yaml; a missing default file is OK.
func Load(path string) (Config, error) {
	cfg := Default()
	explicit := path != ""
	if path == "" {
		if v := os.Getenv("BOT_CONFIG"); v != "" {
			path = v
			explicit = true
		} else {
			path = "configs/bot.yaml"
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && !explicit {
			applyEnv(&cfg)
			normalizeDrivers(&cfg)
			return cfg, nil
		}
		return Config{}, fmt.Errorf("botconfig: read %s: %w", path, err)
	}
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("botconfig: parse %s: %w", path, err)
	}
	applyEnv(&cfg)
	normalizeDrivers(&cfg)
	return cfg, nil
}

func normalizeDrivers(cfg *Config) {
	if strings.TrimSpace(cfg.Placement) == "" {
		switch strings.TrimSpace(cfg.RuntimeMode) {
		case RuntimeModeSplit, PlacementRemote:
			cfg.Placement = PlacementRemote
		default:
			cfg.Placement = PlacementLocal
		}
	}
	if strings.TrimSpace(cfg.RuntimeMode) == "" {
		if cfg.Placement == PlacementRemote {
			cfg.RuntimeMode = RuntimeModeSplit
		} else {
			cfg.RuntimeMode = RuntimeModeAllinone
		}
	}
	if strings.TrimSpace(cfg.Queue.Driver) == "" {
		cfg.Queue.Driver = QueueDriverMemory
	}
	if strings.TrimSpace(cfg.Blob.Driver) == "" {
		cfg.Blob.Driver = BlobDriverLocalFS
	}
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("HTTP_ADDR"); v != "" {
		cfg.HTTPAddr = v
	}
	if v := os.Getenv("DATA_DIR"); v != "" {
		cfg.BlobRoot = strings.TrimRight(v, "/") + "/blob"
		if cfg.DatabaseDSN == "" {
			cfg.DatabaseDSN = strings.TrimRight(v, "/") + "/app.db"
		}
	}
	if v := os.Getenv("COMFYUI_BASE_URL"); v != "" {
		cfg.ComfyUIBaseURL = v
	}
	if v := os.Getenv("COMFY_MOCK"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			// accept 1/0
			switch strings.ToLower(strings.TrimSpace(v)) {
			case "1", "yes", "on":
				b = true
			case "0", "no", "off":
				b = false
			default:
				b = cfg.ComfyMock
			}
		}
		cfg.ComfyMock = b
	}
	if v := os.Getenv("PLACEMENT"); v != "" {
		cfg.Placement = strings.TrimSpace(v)
	}
	if v := os.Getenv("RUNTIME_MODE"); v != "" {
		cfg.RuntimeMode = strings.TrimSpace(v)
	}
	if v := os.Getenv("QUEUE_DRIVER"); v != "" {
		cfg.Queue.Driver = strings.TrimSpace(v)
	}
	if v := os.Getenv("BLOB_DRIVER"); v != "" {
		cfg.Blob.Driver = strings.TrimSpace(v)
	}
	if v := os.Getenv("TOS_ENDPOINT"); v != "" {
		cfg.Blob.TOS.Endpoint = strings.TrimSpace(v)
	}
	if v := os.Getenv("TOS_REGION"); v != "" {
		cfg.Blob.TOS.Region = strings.TrimSpace(v)
	}
	if v := os.Getenv("TOS_BUCKET"); v != "" {
		cfg.Blob.TOS.Bucket = strings.TrimSpace(v)
	}
}
