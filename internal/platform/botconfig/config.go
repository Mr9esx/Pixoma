package botconfig

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	RuntimeModeAllinone = "allinone"
	RuntimeModeSplit    = "split"

	QueueDriverMemory = "memory"
	QueueDriverRedis  = "redis"

	BlobDriverLocalFS = "localfs"
	BlobDriverS3      = "s3"
)

// Config is the bot process configuration.
type Config struct {
	HTTPAddr          string `yaml:"http_addr"`
	DatabaseDSN       string `yaml:"database_dsn"`
	BlobRoot          string `yaml:"blob_root"`
	ComfyUIBaseURL    string `yaml:"comfyui_base_url"`
	DefaultInstanceID string `yaml:"default_instance_id"`
	// ComfyInstances optionally seeds multiple ComfyUI instances on startup.
	// When empty, ComfyUIBaseURL (+ DefaultInstanceID) is upserted instead.
	ComfyInstances   []ComfyInstanceSeed `yaml:"comfy_instances"`
	TelegramBotToken string              `yaml:"telegram_bot_token"`
	CaseSeedDir      string              `yaml:"case_seed_dir"`
	// ComfyMock enables the in-process ComfyUI mock (default true).
	// Set false (or COMFY_MOCK=0) to call a real ComfyUI at ComfyUIBaseURL.
	ComfyMock bool `yaml:"comfy_mock"`
	// HealthProbeInterval is how often enabled real Comfy instances are probed
	// via SystemStats (default 30s). Parsed as Go duration, e.g. "30s".
	HealthProbeInterval string `yaml:"health_probe_interval"`

	// RuntimeMode selects allinone vs split deployment (default allinone).
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
	Driver string `yaml:"driver"` // localfs | s3
}

// ComfyInstanceSeed is one row under comfy_instances in bot YAML.
type ComfyInstanceSeed struct {
	ID           string   `yaml:"id"`
	BaseURL      string   `yaml:"base_url"`
	Enabled      *bool    `yaml:"enabled"`
	Capabilities []string `yaml:"capabilities"`
}

func Default() Config {
	return Config{
		HTTPAddr:            ":8080",
		BlobRoot:            "data/blob",
		ComfyUIBaseURL:      "http://127.0.0.1:8188",
		DefaultInstanceID:   "local",
		CaseSeedDir:         "configs/cases",
		ComfyMock:           true,
		HealthProbeInterval: "30s",
		RuntimeMode:         RuntimeModeAllinone,
		Queue:               QueueConfig{Driver: QueueDriverMemory},
		Blob:                BlobConfig{Driver: BlobDriverLocalFS},
	}
}

// ValidateRuntimeDrivers checks runtime_mode vs queue/blob driver combinations.
func (c Config) ValidateRuntimeDrivers() error {
	mode := strings.TrimSpace(c.RuntimeMode)
	if mode == "" {
		mode = RuntimeModeAllinone
	}
	q := strings.TrimSpace(c.Queue.Driver)
	if q == "" {
		q = QueueDriverMemory
	}
	b := strings.TrimSpace(c.Blob.Driver)
	if b == "" {
		b = BlobDriverLocalFS
	}
	switch mode {
	case RuntimeModeAllinone:
		if q != QueueDriverMemory {
			return fmt.Errorf("botconfig: allinone requires queue.driver=memory, got %q", q)
		}
		if b != BlobDriverLocalFS {
			return fmt.Errorf("botconfig: allinone requires blob.driver=localfs, got %q", b)
		}
	case RuntimeModeSplit:
		if q != QueueDriverRedis {
			return fmt.Errorf("botconfig: split requires queue.driver=redis, got %q", q)
		}
		if b != BlobDriverS3 {
			return fmt.Errorf("botconfig: split requires blob.driver=s3, got %q", b)
		}
	default:
		return fmt.Errorf("botconfig: unknown runtime_mode %q", mode)
	}
	return nil
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
	if strings.TrimSpace(cfg.RuntimeMode) == "" {
		cfg.RuntimeMode = RuntimeModeAllinone
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
	if v := os.Getenv("INSTANCE_ID"); v != "" {
		cfg.DefaultInstanceID = v
	}
	if v := os.Getenv("TG_BOT_TOKEN"); v != "" {
		cfg.TelegramBotToken = v
	}
	if v := os.Getenv("TELEGRAM_BOT_TOKEN"); v != "" {
		cfg.TelegramBotToken = v
	}
	if v := os.Getenv("CASE_SEED_DIR"); v != "" {
		cfg.CaseSeedDir = v
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
	if v := os.Getenv("RUNTIME_MODE"); v != "" {
		cfg.RuntimeMode = strings.TrimSpace(v)
	}
	if v := os.Getenv("QUEUE_DRIVER"); v != "" {
		cfg.Queue.Driver = strings.TrimSpace(v)
	}
	if v := os.Getenv("BLOB_DRIVER"); v != "" {
		cfg.Blob.Driver = strings.TrimSpace(v)
	}
}
