package persistence_test

import (
	"context"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:comfy_instance_test_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &persistence.EdgeRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return gdb
}

func TestInstanceRepo_UpsertGetList(t *testing.T) {
	gdb := openTestDB(t)
	repo := persistence.NewEdgeRepository(gdb)
	ctx := context.Background()

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
	if len(got.Capabilities) != 1 || got.Capabilities[0] != "sdxl" {
		t.Fatalf("capabilities=%v", got.Capabilities)
	}

	rec.Capabilities = []string{"sdxl", "video"}
	rec.UpdatedAt = now.Add(time.Second)
	if err := repo.Upsert(ctx, rec); err != nil {
		t.Fatalf("upsert update: %v", err)
	}

	// Simulate restart: new repository on same DB.
	repo2 := persistence.NewEdgeRepository(gdb)
	if _, err := repo2.Get(ctx, "gpu-1"); err != nil {
		t.Fatalf("get after restart: %v", err)
	}

	if err := repo2.Upsert(ctx, &edge.Record{
		ID:        "gpu-2",
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

func TestInstanceRepo_UpdatePresenceInfo(t *testing.T) {
	gdb := openTestDB(t)
	repo := persistence.NewEdgeRepository(gdb)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	if err := repo.Upsert(ctx, &edge.Record{
		ID:        "gpu-pres",
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	startedAt := now.Add(-time.Hour)
	if err := repo.UpdatePresenceInfo(ctx, "gpu-pres", &startedAt, "v0.1.0"); err != nil {
		t.Fatalf("update presence info: %v", err)
	}
	got, err := repo.Get(ctx, "gpu-pres")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.StartedAt == nil || !got.StartedAt.Equal(startedAt) {
		t.Fatalf("started_at=%v want %v", got.StartedAt, startedAt)
	}
	if got.ComfyVersion != "v0.1.0" {
		t.Fatalf("comfy_version=%q", got.ComfyVersion)
	}

	// Empty version must not wipe the previously stored value.
	if err := repo.UpdatePresenceInfo(ctx, "gpu-pres", nil, ""); err != nil {
		t.Fatalf("update presence info empty: %v", err)
	}
	again, err := repo.Get(ctx, "gpu-pres")
	if err != nil {
		t.Fatalf("get after empty update: %v", err)
	}
	if again.ComfyVersion != "v0.1.0" {
		t.Fatalf("empty version wiped stored value: %q", again.ComfyVersion)
	}
	if again.StartedAt == nil || !again.StartedAt.Equal(startedAt) {
		t.Fatalf("nil started_at must keep previous value: %v", again.StartedAt)
	}
}

func TestInstanceRepo_HardwareRoundTrip(t *testing.T) {
	gdb := openTestDB(t)
	repo := persistence.NewEdgeRepository(gdb)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	rec := &edge.Record{
		ID:        sharedkernel.EdgeID("gpu-hw"),
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
		Hardware: edge.Hardware{
			CPUModel:    "Intel",
			CPUCores:    8,
			RAMBytes:    16 << 30,
			GPUs:        []edge.GPU{{Name: "Fake GPU", VRAMBytes: 8 << 30}},
			CollectedAt: now,
		},
		HardwareRefreshRequested: true,
	}
	if err := repo.Upsert(ctx, rec); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got, err := repo.Get(ctx, "gpu-hw")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Hardware.CPUModel != "Intel" || got.Hardware.CPUCores != 8 || got.Hardware.RAMBytes != 16<<30 {
		t.Fatalf("hardware=%+v", got.Hardware)
	}
	if len(got.Hardware.GPUs) != 1 || got.Hardware.GPUs[0].VRAMBytes != 8<<30 {
		t.Fatalf("gpus=%+v", got.Hardware.GPUs)
	}
	if !got.HardwareRefreshRequested {
		t.Fatal("refresh flag not stored")
	}

	rec.Hardware = edge.Hardware{}
	rec.HardwareRefreshRequested = false
	if err := repo.Upsert(ctx, rec); err != nil {
		t.Fatalf("upsert update: %v", err)
	}
	again, err := repo.Get(ctx, "gpu-hw")
	if err != nil {
		t.Fatalf("get after update: %v", err)
	}
	if again.Hardware.CPUModel != "Intel" {
		t.Fatalf("seed upsert wiped hardware: %+v", again.Hardware)
	}
	if !again.HardwareRefreshRequested {
		t.Fatal("seed upsert wiped refresh flag")
	}
}
