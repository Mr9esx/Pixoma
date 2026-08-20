package appboot_test

import (
	"context"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/appboot"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestBootstrap_MigratesInstancesAndAllowsRoundTrip(t *testing.T) {
	ctx := context.Background()
	dsn := "file:appboot_" + t.Name() + "?mode=memory&cache=shared"

	gdb, cleanup, err := appboot.Bootstrap(ctx, appboot.Options{
		DSN:          dsn,
		MigrateEdges: true,
	})
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	t.Cleanup(func() {
		if cleanup != nil {
			_ = cleanup()
		}
	})

	repo := persistence.NewEdgeRepository(gdb)
	now := time.Now().UTC().Truncate(time.Second)
	rec := &edge.Record{
		ID:           sharedkernel.EdgeID("gpu-1"),
		Enabled:      true,
		Capabilities: []string{"sdxl"},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := repo.Upsert(ctx, rec); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	got, err := repo.Get(ctx, "gpu-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !got.Enabled {
		t.Fatalf("get mismatch: %+v", got)
	}
}

func TestBootstrap_SeedsInstancesFromOptions(t *testing.T) {
	ctx := context.Background()
	dsn := "file:appboot_" + t.Name() + "?mode=memory&cache=shared"
	enabled := true

	gdb, cleanup, err := appboot.Bootstrap(ctx, appboot.Options{
		DSN:          dsn,
		MigrateEdges: true,
		Seed: &edge.SeedConfig{
			Edges: []edge.SeedInstance{{
				ID:           "seed-1",
				Enabled:      &enabled,
				Capabilities: []string{"mock"},
			}},
		},
	})
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	t.Cleanup(func() {
		if cleanup != nil {
			_ = cleanup()
		}
	})

	repo := persistence.NewEdgeRepository(gdb)
	got, err := repo.Get(ctx, "seed-1")
	if err != nil {
		t.Fatalf("get seeded: %v", err)
	}
	if got.ID != "seed-1" || !got.Enabled {
		t.Fatalf("seeded=%+v", got)
	}
}

func TestBootstrap_WithoutSeedInsertsNoEdges(t *testing.T) {
	ctx := context.Background()
	dsn := "file:appboot_" + t.Name() + "?mode=memory&cache=shared"

	gdb, cleanup, err := appboot.Bootstrap(ctx, appboot.Options{
		DSN:          dsn,
		MigrateEdges: true,
	})
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	t.Cleanup(func() {
		if cleanup != nil {
			_ = cleanup()
		}
	})

	repo := persistence.NewEdgeRepository(gdb)
	rows, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected no seeded edges, got %d: %+v", len(rows), rows)
	}
}
