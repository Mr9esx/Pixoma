package instance

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// ErrNotFound is returned when an instance id is missing.
var ErrNotFound = errors.New("instance: not found")

// Repository persists Comfy instance metadata.
type Repository interface {
	Upsert(ctx context.Context, r *Record) error
	Get(ctx context.Context, id sharedkernel.InstanceID) (*Record, error)
	List(ctx context.Context) ([]*Record, error)
	Delete(ctx context.Context, id sharedkernel.InstanceID) error
}

// SeedConfig drives startup upsert of instance rows from bot config.
type SeedConfig struct {
	ComfyInstances    []SeedInstance
	DefaultInstanceID string
	ComfyUIBaseURL    string
	ComfyMock         bool
}

// SeedInstance is one config-file seed row.
type SeedInstance struct {
	ID           string   `yaml:"id"`
	BaseURL      string   `yaml:"base_url"`
	Enabled      *bool    `yaml:"enabled"`
	Capabilities []string `yaml:"capabilities"`
}

// SeedFromConfig upserts instances from comfy_instances or a single comfyui_base_url.
// When ComfyMock is true and there is no explicit list, still upserts the default instance.
func SeedFromConfig(ctx context.Context, repo Repository, cfg SeedConfig) (int, error) {
	if repo == nil {
		return 0, fmt.Errorf("instance: nil repository")
	}
	now := time.Now().UTC()
	n := 0

	if len(cfg.ComfyInstances) > 0 {
		for _, s := range cfg.ComfyInstances {
			if s.ID == "" || s.BaseURL == "" {
				return n, fmt.Errorf("instance: seed entry requires id and base_url")
			}
			enabled := true
			if s.Enabled != nil {
				enabled = *s.Enabled
			}
			rec := &Record{
				ID:           sharedkernel.InstanceID(s.ID),
				BaseURL:      s.BaseURL,
				Enabled:      enabled,
				Capabilities: append([]string(nil), s.Capabilities...),
				CreatedAt:    now,
				UpdatedAt:    now,
			}
			if err := repo.Upsert(ctx, rec); err != nil {
				return n, err
			}
			n++
		}
		return n, nil
	}

	if cfg.ComfyUIBaseURL == "" && !cfg.ComfyMock {
		return 0, nil
	}
	id := cfg.DefaultInstanceID
	if id == "" {
		id = "local"
	}
	baseURL := cfg.ComfyUIBaseURL
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8188"
	}
	rec := &Record{
		ID:        sharedkernel.InstanceID(id),
		BaseURL:   baseURL,
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.Upsert(ctx, rec); err != nil {
		return 0, err
	}
	return 1, nil
}
