package adminconfig

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the admin-api process configuration (no Telegram fields).
type Config struct {
	HTTPAddr    string   `yaml:"http_addr"`
	DatabaseDSN string   `yaml:"database_dsn"`
	CORSOrigins []string `yaml:"cors_origins"`
	// ComfyMock aligns observation paths with bot (COMFY_MOCK).
	ComfyMock bool `yaml:"comfy_mock"`
}

func Default() Config {
	return Config{
		HTTPAddr: "127.0.0.1:8081",
		CORSOrigins: []string{
			"http://localhost:5173",
			"http://127.0.0.1:5173",
			"http://localhost:3000",
			"http://127.0.0.1:3000",
		},
		ComfyMock: true,
	}
}

// Load reads optional YAML then applies env overrides.
// Empty path tries ADMIN_CONFIG, then configs/admin-api.yaml; a missing default file is OK.
func Load(path string) (Config, error) {
	cfg := Default()
	explicit := path != ""
	if path == "" {
		if v := os.Getenv("ADMIN_CONFIG"); v != "" {
			path = v
			explicit = true
		} else {
			path = "configs/admin-api.yaml"
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && !explicit {
			applyEnv(&cfg)
			return cfg, nil
		}
		return Config{}, fmt.Errorf("adminconfig: read %s: %w", path, err)
	}
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("adminconfig: parse %s: %w", path, err)
	}
	if len(cfg.CORSOrigins) == 0 {
		cfg.CORSOrigins = Default().CORSOrigins
	}
	applyEnv(&cfg)
	return cfg, nil
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("HTTP_ADDR"); v != "" {
		cfg.HTTPAddr = v
	}
	if v := os.Getenv("DATA_DIR"); v != "" {
		if cfg.DatabaseDSN == "" {
			cfg.DatabaseDSN = strings.TrimRight(v, "/") + "/app.db"
		}
	}
	if v := os.Getenv("DATABASE_DSN"); v != "" {
		cfg.DatabaseDSN = v
	}
	if v := os.Getenv("COMFY_MOCK"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
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
}
