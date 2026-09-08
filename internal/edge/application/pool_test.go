package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/edge/application"
	edgedomain "github.com/Mr9esx/Pixoma/internal/edge/domain"
	"github.com/Mr9esx/Pixoma/internal/edge/infrastructure/persistence"
	"github.com/Mr9esx/Pixoma/internal/platform/db"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

func TestPool_RefreshAndHealthyFilter(t *testing.T) {
	dsn := "file:comfy_pool_test_" + t.Name() + "?mode=memory&cache=shared"
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

	if err := repo.Upsert(ctx, &edgedomain.Record{
		ID: "gpu-1", Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Upsert(ctx, &edgedomain.Record{
		ID: "gpu-2", Enabled: false,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	pool := application.NewPool(repo, application.PoolOptions{})
	if err := pool.Refresh(ctx); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	pool.SetHealthy("gpu-1", true)
	healthy, err := pool.ListHealthy(ctx, edgedomain.CapabilityFilter{})
	if err != nil {
		t.Fatalf("list healthy: %v", err)
	}
	if len(healthy) != 1 || healthy[0].ID != "gpu-1" {
		t.Fatalf("healthy=%v", healthy)
	}

	// Disabled remains excluded even if marked healthy.
	pool.SetHealthy("gpu-2", true)
	healthy, err = pool.ListHealthy(ctx, edgedomain.CapabilityFilter{})
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range healthy {
		if h.ID == sharedkernel.EdgeID("gpu-2") {
			t.Fatal("disabled gpu-2 must not appear in ListHealthy")
		}
	}
}

func TestSeedFromConfig_OnlyExplicitEdges(t *testing.T) {
	dsn := "file:comfy_seed_test_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &persistence.EdgeRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := persistence.NewEdgeRepository(gdb)
	ctx := context.Background()

	// 没有显式 comfy_instances 时不再插入默认节点。
	n, err := edgedomain.SeedFromConfig(ctx, repo, edgedomain.SeedConfig{})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if n != 0 {
		t.Fatalf("seeded=%d", n)
	}
	rows, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected no rows, got %d", len(rows))
	}

	// 显式列表仍然按配置种子插入。
	enabled := true
	n, err = edgedomain.SeedFromConfig(ctx, repo, edgedomain.SeedConfig{
		Edges: []edgedomain.SeedInstance{{ID: "gpu-1", Enabled: &enabled}},
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if n != 1 {
		t.Fatalf("seeded=%d", n)
	}
	got, err := repo.Get(ctx, "gpu-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !got.Enabled {
		t.Fatalf("got=%+v", got)
	}
}

func TestPool_AfterRefreshCalledWithInstances(t *testing.T) {
	dsn := "file:comfy_after_refresh_" + t.Name() + "?mode=memory&cache=shared"
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

	if err := repo.Upsert(ctx, &edgedomain.Record{
		ID: "gpu-1", Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	var seen []sharedkernel.EdgeID
	pool := application.NewPool(repo, application.PoolOptions{})
	pool.SetAfterRefresh(func(instances []edgedomain.Instance) {
		seen = nil
		for _, inst := range instances {
			seen = append(seen, inst.ID)
		}
	})
	if err := pool.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 1 || seen[0] != "gpu-1" {
		t.Fatalf("after first refresh seen=%v", seen)
	}

	if err := repo.Upsert(ctx, &edgedomain.Record{
		ID: "gpu-new", Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := pool.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 2 {
		t.Fatalf("after second refresh seen=%v want 2 ids", seen)
	}
	found := false
	for _, id := range seen {
		if id == "gpu-new" {
			found = true
		}
	}
	if !found {
		t.Fatalf("seen=%v missing gpu-new", seen)
	}
}
