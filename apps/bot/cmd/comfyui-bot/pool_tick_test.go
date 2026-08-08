package main

import (
	"context"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestTickPool_RefreshesNewEnabledInstanceFromDB(t *testing.T) {
	dsn := "file:bot_tick_pool_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &persistence.InstanceRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := persistence.NewInstanceRepository(gdb)
	ctx := context.Background()
	now := time.Now().UTC()

	if err := repo.Upsert(ctx, &instance.Record{
		ID: "gpu-1", BaseURL: "http://127.0.0.1:8188", Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	pool := instance.NewPool(repo, instance.PoolOptions{Mock: true})
	if err := pool.Refresh(ctx); err != nil {
		t.Fatalf("initial refresh: %v", err)
	}
	if len(pool.List()) != 1 {
		t.Fatalf("initial list=%v", pool.List())
	}

	if err := repo.Upsert(ctx, &instance.Record{
		ID: "gpu-new", BaseURL: "http://127.0.0.1:8190", Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	// Admin wrote to DB; bot has not refreshed yet.
	for _, inst := range pool.List() {
		if inst.ID == sharedkernel.InstanceID("gpu-new") {
			t.Fatal("gpu-new must not appear before tickPool")
		}
	}

	tickPool(ctx, pool)

	found := false
	for _, inst := range pool.List() {
		if inst.ID == sharedkernel.InstanceID("gpu-new") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("after tickPool list=%v missing gpu-new", pool.List())
	}
	if _, err := pool.Client("gpu-new"); err != nil {
		t.Fatalf("client for gpu-new: %v", err)
	}
}
