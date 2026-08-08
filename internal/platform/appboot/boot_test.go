package appboot_test

import (
	"context"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/appboot"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestBootstrap_MigratesInstancesAndAllowsRoundTrip(t *testing.T) {
	ctx := context.Background()
	dsn := "file:appboot_" + t.Name() + "?mode=memory&cache=shared"

	gdb, cleanup, err := appboot.Bootstrap(ctx, appboot.Options{
		DSN:              dsn,
		MigrateInstances: true,
	})
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	t.Cleanup(func() {
		if cleanup != nil {
			_ = cleanup()
		}
	})

	repo := persistence.NewInstanceRepository(gdb)
	now := time.Now().UTC().Truncate(time.Second)
	rec := &instance.Record{
		ID:           sharedkernel.InstanceID("gpu-1"),
		BaseURL:      "http://127.0.0.1:8188",
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
	if got.BaseURL != rec.BaseURL || !got.Enabled {
		t.Fatalf("get mismatch: %+v", got)
	}
}

func TestBootstrap_SeedsInstancesFromOptions(t *testing.T) {
	ctx := context.Background()
	dsn := "file:appboot_" + t.Name() + "?mode=memory&cache=shared"
	enabled := true

	gdb, cleanup, err := appboot.Bootstrap(ctx, appboot.Options{
		DSN:              dsn,
		MigrateInstances: true,
		Seed: &instance.SeedConfig{
			ComfyInstances: []instance.SeedInstance{{
				ID:           "seed-1",
				BaseURL:      "http://127.0.0.1:9000",
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

	repo := persistence.NewInstanceRepository(gdb)
	got, err := repo.Get(ctx, "seed-1")
	if err != nil {
		t.Fatalf("get seeded: %v", err)
	}
	if got.BaseURL != "http://127.0.0.1:9000" {
		t.Fatalf("base_url=%q", got.BaseURL)
	}
}
