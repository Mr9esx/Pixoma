package instance_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/comfyui"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestPool_RefreshAndHealthyFilter(t *testing.T) {
	dsn := "file:comfy_pool_test_" + t.Name() + "?mode=memory&cache=shared"
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
	if err := repo.Upsert(ctx, &instance.Record{
		ID: "gpu-2", BaseURL: "http://127.0.0.1:8189", Enabled: false,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	pool := instance.NewPool(repo, instance.PoolOptions{Mock: false})
	if err := pool.Refresh(ctx); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	pool.SetHealthy("gpu-1", true)
	healthy, err := pool.ListHealthy(ctx, instance.CapabilityFilter{})
	if err != nil {
		t.Fatalf("list healthy: %v", err)
	}
	if len(healthy) != 1 || healthy[0].ID != "gpu-1" {
		t.Fatalf("healthy=%v", healthy)
	}

	// Update base_url and refresh; Client must point at new URL.
	if err := repo.Upsert(ctx, &instance.Record{
		ID: "gpu-1", BaseURL: "http://127.0.0.1:9191", Enabled: true,
		CreatedAt: now, UpdatedAt: now.Add(time.Second),
	}); err != nil {
		t.Fatal(err)
	}
	if err := pool.Refresh(ctx); err != nil {
		t.Fatalf("refresh after url change: %v", err)
	}
	cli, err := pool.Client("gpu-1")
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	httpCli, ok := cli.(*comfyui.HTTP)
	if !ok {
		t.Fatalf("client type %T", cli)
	}
	if httpCli.BaseURL != "http://127.0.0.1:9191" {
		t.Fatalf("client base_url=%q", httpCli.BaseURL)
	}

	// Disabled remains excluded even if marked healthy.
	pool.SetHealthy("gpu-2", true)
	healthy, err = pool.ListHealthy(ctx, instance.CapabilityFilter{})
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range healthy {
		if h.ID == sharedkernel.InstanceID("gpu-2") {
			t.Fatal("disabled gpu-2 must not appear in ListHealthy")
		}
	}
}

func TestSeedFromConfig_SingleBaseURL(t *testing.T) {
	dsn := "file:comfy_seed_test_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &persistence.InstanceRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := persistence.NewInstanceRepository(gdb)
	ctx := context.Background()

	n, err := instance.SeedFromConfig(ctx, repo, instance.SeedConfig{
		DefaultInstanceID: "local",
		ComfyUIBaseURL:    "http://127.0.0.1:8188",
		ComfyMock:         true,
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if n != 1 {
		t.Fatalf("seeded=%d", n)
	}
	got, err := repo.Get(ctx, "local")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.BaseURL != "http://127.0.0.1:8188" || !got.Enabled {
		t.Fatalf("got=%+v", got)
	}
}

func TestPool_ProbeMarksUnhealthy(t *testing.T) {
	dsn := "file:comfy_probe_test_" + t.Name() + "?mode=memory&cache=shared"
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

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	if err := repo.Upsert(ctx, &instance.Record{
		ID: "gpu-down", BaseURL: srv.URL, Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	pool := instance.NewPool(repo, instance.PoolOptions{Mock: false})
	if err := pool.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	pool.SetHealthy("gpu-down", true)
	pool.Probe(ctx)

	healthy, err := pool.ListHealthy(ctx, instance.CapabilityFilter{})
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range healthy {
		if h.ID == "gpu-down" {
			t.Fatal("unreachable instance must leave ListHealthy")
		}
	}
}
