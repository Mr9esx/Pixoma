package edge

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// ErrNotFound is returned when an instance id is missing.
var ErrNotFound = errors.New("edge: not found")

// Repository persists Comfy instance metadata.
type Repository interface {
	Upsert(ctx context.Context, r *Record) error
	Get(ctx context.Context, id sharedkernel.EdgeID) (*Record, error)
	List(ctx context.Context) ([]*Record, error)
	Delete(ctx context.Context, id sharedkernel.EdgeID) error
	UpdateAgentTokenEnc(ctx context.Context, id sharedkernel.EdgeID, enc string) error
	UpdateHardware(ctx context.Context, id sharedkernel.EdgeID, hw Hardware) error
	SetHardwareRefreshRequested(ctx context.Context, id sharedkernel.EdgeID, requested bool) error
	UpdatePresenceInfo(ctx context.Context, id sharedkernel.EdgeID, startedAt *time.Time, comfyVersion string) error
}

// SeedConfig drives startup upsert of instance rows from bot config.
type SeedConfig struct {
	Edges         []SeedInstance
	DefaultEdgeID string
	ComfyMock     bool
}

// SeedInstance is one config-file seed row.
type SeedInstance struct {
	ID           string   `yaml:"id"`
	Enabled      *bool    `yaml:"enabled"`
	Capabilities []string `yaml:"capabilities"`
}

// SeedFromConfig upserts instances from comfy_instances or a single default edge.
func SeedFromConfig(ctx context.Context, repo Repository, cfg SeedConfig) (int, error) {
	if repo == nil {
		return 0, fmt.Errorf("edge: nil repository")
	}
	now := time.Now().UTC()
	n := 0

	if len(cfg.Edges) > 0 {
		for _, s := range cfg.Edges {
			if s.ID == "" {
				return n, fmt.Errorf("edge: seed entry requires id")
			}
			enabled := true
			if s.Enabled != nil {
				enabled = *s.Enabled
			}
			rec := &Record{
				ID:           sharedkernel.EdgeID(s.ID),
				Name:         s.ID,
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

	id := cfg.DefaultEdgeID
	if id == "" {
		if !cfg.ComfyMock {
			return 0, nil
		}
		id = "local"
	}
	rec := &Record{
		ID:        sharedkernel.EdgeID(id),
		Name:      id,
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.Upsert(ctx, rec); err != nil {
		return 0, err
	}
	return 1, nil
}
