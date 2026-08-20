package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge/persistence"
)

func TestMetricsRepository_AppendAndList(t *testing.T) {
	dsn := "file:metrics_test_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &persistence.MetricsRow{}); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewMetricsRepository(gdb, 24*time.Hour)
	ctx := context.Background()
	base := time.Now().UTC().Add(-2 * time.Hour)
	for i := 0; i < 3; i++ {
		m := edge.Metrics{CPUUsagePercent: float64(i), CollectedAt: base.Add(time.Duration(i) * time.Minute)}
		if err := repo.Append(ctx, "e1", m); err != nil {
			t.Fatal(err)
		}
	}
	series, err := repo.ListSince(ctx, "e1", base.Add(-time.Hour), 720)
	if err != nil {
		t.Fatal(err)
	}
	if len(series) != 3 || series[0].CPUUsagePercent != 0 || series[2].CPUUsagePercent != 2 {
		t.Fatalf("series: %+v", series)
	}
	limited, err := repo.ListSince(ctx, "e1", base.Add(-time.Hour), 2)
	if err != nil || len(limited) != 2 {
		t.Fatalf("limit: %v %v", limited, err)
	}
}

func TestMetricsRepository_PrunesOldRows(t *testing.T) {
	dsn := "file:metrics_test_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &persistence.MetricsRow{}); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewMetricsRepository(gdb, time.Hour)
	ctx := context.Background()
	now := time.Now().UTC()
	old := edge.Metrics{CPUUsagePercent: 1, CollectedAt: now.Add(-2 * time.Hour)}
	if err := repo.Append(ctx, "e1", old); err != nil {
		t.Fatal(err)
	}
	fresh := edge.Metrics{CPUUsagePercent: 2, CollectedAt: now}
	if err := repo.Append(ctx, "e1", fresh); err != nil {
		t.Fatal(err)
	}
	series, err := repo.ListSince(ctx, "e1", now.Add(-3*time.Hour), 720)
	if err != nil {
		t.Fatal(err)
	}
	if len(series) != 1 || series[0].CPUUsagePercent != 2 {
		t.Fatalf("want only fresh row, got %+v", series)
	}
}
