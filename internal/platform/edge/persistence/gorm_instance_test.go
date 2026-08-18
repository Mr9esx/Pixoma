package persistence_test

import (
	"context"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:comfy_instance_test_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &persistence.InstanceRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return gdb
}

func TestInstanceRepo_UpsertGetList(t *testing.T) {
	gdb := openTestDB(t)
	repo := persistence.NewInstanceRepository(gdb)
	ctx := context.Background()

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
	if len(got.Capabilities) != 1 || got.Capabilities[0] != "sdxl" {
		t.Fatalf("capabilities=%v", got.Capabilities)
	}

	rec.BaseURL = "http://127.0.0.1:8190"
	rec.Capabilities = []string{"sdxl", "video"}
	rec.UpdatedAt = now.Add(time.Second)
	if err := repo.Upsert(ctx, rec); err != nil {
		t.Fatalf("upsert update: %v", err)
	}

	// Simulate restart: new repository on same DB.
	repo2 := persistence.NewInstanceRepository(gdb)
	again, err := repo2.Get(ctx, "gpu-1")
	if err != nil {
		t.Fatalf("get after restart: %v", err)
	}
	if again.BaseURL != "http://127.0.0.1:8190" {
		t.Fatalf("base_url after restart=%q", again.BaseURL)
	}

	if err := repo2.Upsert(ctx, &instance.Record{
		ID:        "gpu-2",
		BaseURL:   "http://127.0.0.1:8189",
		Enabled:   false,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("upsert gpu-2: %v", err)
	}
	list, err := repo2.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("list len=%d", len(list))
	}

	if err := repo2.Delete(ctx, "gpu-2"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo2.Get(ctx, "gpu-2"); err == nil {
		t.Fatal("expected not found after delete")
	}
}
