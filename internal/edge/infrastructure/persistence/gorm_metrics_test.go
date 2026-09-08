package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/platform/db"
	edge "github.com/Mr9esx/Pixoma/internal/edge/domain"
	"github.com/Mr9esx/Pixoma/internal/edge/infrastructure/persistence"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
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

func TestMetricsRepository_LatestAll(t *testing.T) {
	dsn := "file:metrics_latest_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &persistence.MetricsRow{}); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewMetricsRepository(gdb, 24*time.Hour)
	ctx := context.Background()
	base := time.Now().UTC().Add(-time.Hour)
	for _, tc := range []struct {
		edge string
		cpu  float64
		at   time.Time
	}{
		{"e1", 10, base},
		{"e1", 40, base.Add(5 * time.Minute)},
		{"e2", 25, base.Add(2 * time.Minute)},
	} {
		if err := repo.Append(ctx, sharedkernel.EdgeID(tc.edge), edge.Metrics{CPUUsagePercent: tc.cpu, CollectedAt: tc.at}); err != nil {
			t.Fatal(err)
		}
	}
	latest, err := repo.LatestAll(ctx, base.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(latest) != 2 || latest["e1"].CPUUsagePercent != 40 || latest["e2"].CPUUsagePercent != 25 {
		t.Fatalf("latest: %+v", latest)
	}
}
