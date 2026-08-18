package main

import (
	"context"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestTickPool_RefreshesNewEnabledInstanceFromDB(t *testing.T) {
	dsn := "file:bot_tick_pool_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &persistence.EdgeRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := persistence.NewEdgeRepository(gdb)
	ctx := context.Background()
	now := time.Now().UTC()

	if err := repo.Upsert(ctx, &edge.Record{
		ID: "gpu-1", Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	pool := edge.NewPool(repo, edge.PoolOptions{})
	if err := pool.Refresh(ctx); err != nil {
		t.Fatalf("initial refresh: %v", err)
	}
	if len(pool.List()) != 1 {
		t.Fatalf("initial list=%v", pool.List())
	}

	if err := repo.Upsert(ctx, &edge.Record{
		ID: "gpu-new", Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	// Admin wrote to DB; bot has not refreshed yet.
	for _, inst := range pool.List() {
		if inst.ID == sharedkernel.EdgeID("gpu-new") {
			t.Fatal("gpu-new must not appear before tickPool")
		}
	}

	tickPool(ctx, pool)

	found := false
	for _, inst := range pool.List() {
		if inst.ID == sharedkernel.EdgeID("gpu-new") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("after tickPool list=%v missing gpu-new", pool.List())
	}
}
