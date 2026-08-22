package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/taskstats"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/taskstats/persistence"
)

func TestGormStatsRepository_AddTerminalAndList(t *testing.T) {
	dsn := "file:stats_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb,
		&persistence.DailyStatsRow{},
		&persistence.EdgeDailyStatsRow{},
		&persistence.ErrorDailyStatsRow{},
		&persistence.CaseDailyStatsRow{},
	); err != nil {
		t.Fatal(err)
	}
	loc := time.FixedZone("CST", 8*3600)
	repo := persistence.NewGormStatsRepository(gdb, 365*24*time.Hour, loc)
	ctx := context.Background()
	base := time.Date(2026, 8, 20, 23, 30, 0, 0, time.UTC) // 08-21 07:30 CST
	in := taskstats.AddTerminalInput{
		EdgeID: "gpu-1", ErrorCode: "timeout", Status: taskstats.StatusFailed,
		CompletedAt: base, CreatedAt: base.Add(-2 * time.Minute),
		CaseID: 1, QueueDurationMS: 30000, ExecDurationMS: 90000,
	}
	if err := repo.AddTerminal(ctx, in); err != nil {
		t.Fatal(err)
	}
	if err := repo.AddTerminal(ctx, taskstats.AddTerminalInput{
		Status: taskstats.StatusSucceeded, CompletedAt: base, CreatedAt: base,
	}); err != nil {
		t.Fatal(err)
	}
	days, err := repo.ListDaily(ctx, "2026-08-20", "2026-08-21")
	if err != nil {
		t.Fatal(err)
	}
	if len(days) != 1 || days[0].Date != "2026-08-21" || days[0].Processed != 2 ||
		days[0].Failed != 1 || days[0].Succeeded != 1 || days[0].TotalDurationMS != 120000 {
		t.Fatalf("daily: %+v", days)
	}
	if days[0].TotalQueueMS != 30000 || days[0].TotalExecMS != 90000 {
		t.Fatalf("daily queue/exec: %+v", days[0])
	}
	errs, err := repo.ListErrors(ctx, "2026-08-20", "2026-08-21", 10)
	if err != nil || len(errs) != 1 || errs[0].ErrorCode != "timeout" || errs[0].Count != 1 {
		t.Fatalf("errors: %+v %v", errs, err)
	}
	edges, err := repo.ListEdges(ctx, "2026-08-20", "2026-08-21")
	if err != nil || len(edges) != 1 || edges[0].EdgeID != "gpu-1" || edges[0].Count != 1 {
		t.Fatalf("edges: %+v %v", edges, err)
	}
	if edges[0].Succeeded != 0 || edges[0].Failed != 1 {
		t.Fatalf("edges success/fail: %+v", edges[0])
	}
	cases, err := repo.ListCases(ctx, "2026-08-20", "2026-08-21", 10)
	if err != nil || len(cases) != 1 || cases[0].CaseID != 1 || cases[0].Count != 1 || cases[0].TotalDurationMS != 120000 {
		t.Fatalf("cases: %+v %v", cases, err)
	}
}

func TestGormStatsRepository_Prune(t *testing.T) {
	dsn := "file:stats_prune_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &persistence.DailyStatsRow{}, &persistence.EdgeDailyStatsRow{}, &persistence.ErrorDailyStatsRow{}, &persistence.CaseDailyStatsRow{}); err != nil {
		t.Fatal(err)
	}
	loc := time.UTC
	repo := persistence.NewGormStatsRepository(gdb, 24*time.Hour, loc)
	ctx := context.Background()
	now := time.Now().UTC()
	for _, in := range []taskstats.AddTerminalInput{
		{Status: taskstats.StatusSucceeded, CompletedAt: now.Add(-48 * time.Hour), CreatedAt: now.Add(-48 * time.Hour), CaseID: 9},
		{Status: taskstats.StatusSucceeded, CompletedAt: now, CreatedAt: now, CaseID: 9},
	} {
		if err := repo.AddTerminal(ctx, in); err != nil {
			t.Fatal(err)
		}
	}
	days, err := repo.ListDaily(ctx, taskstats.DateOf(now.Add(-72*time.Hour), loc), taskstats.DateOf(now, loc))
	if err != nil {
		t.Fatal(err)
	}
	if len(days) != 1 || days[0].Processed != 1 {
		t.Fatalf("expected only today's row after prune: %+v", days)
	}
}

func TestGormStatsRepository_SkipEmptyDimensions(t *testing.T) {
	dsn := "file:stats_dim_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &persistence.DailyStatsRow{}, &persistence.EdgeDailyStatsRow{}, &persistence.ErrorDailyStatsRow{}, &persistence.CaseDailyStatsRow{}); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewGormStatsRepository(gdb, 365*24*time.Hour, time.UTC)
	ctx := context.Background()
	now := time.Now().UTC()
	in := taskstats.AddTerminalInput{Status: taskstats.StatusCancelled, CompletedAt: now, CreatedAt: now, CaseID: 2}
	for i := 0; i < 2; i++ {
		if err := repo.AddTerminal(ctx, in); err != nil {
			t.Fatal(err)
		}
	}
	edges, err := repo.ListEdges(ctx, taskstats.DateOf(now, time.UTC), taskstats.DateOf(now, time.UTC))
	if err != nil || len(edges) != 0 {
		t.Fatalf("edges should be empty: %+v %v", edges, err)
	}
	errs, err := repo.ListErrors(ctx, taskstats.DateOf(now, time.UTC), taskstats.DateOf(now, time.UTC), 10)
	if err != nil || len(errs) != 0 {
		t.Fatalf("errors should be empty: %+v %v", errs, err)
	}
	days, err := repo.ListDaily(ctx, taskstats.DateOf(now, time.UTC), taskstats.DateOf(now, time.UTC))
	if err != nil || len(days) != 1 || days[0].Cancelled != 2 {
		t.Fatalf("daily incremental: %+v %v", days, err)
	}
	cases, err := repo.ListCases(ctx, taskstats.DateOf(now, time.UTC), taskstats.DateOf(now, time.UTC), 10)
	if err != nil || len(cases) != 1 || cases[0].CaseID != 2 || cases[0].Count != 2 {
		t.Fatalf("cases should count cancelled without edge: %+v %v", cases, err)
	}
}

func TestDateOfBoundaries(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	cases := []struct {
		name string
		in   time.Time
		want string
	}{
		{"utc evening maps to next cst day", time.Date(2026, 8, 20, 23, 59, 59, 999999999, time.UTC), "2026-08-21"},
		{"utc early maps to same cst day", time.Date(2026, 8, 20, 15, 59, 59, 999999999, time.UTC), "2026-08-20"},
		{"cst midnight", time.Date(2026, 8, 21, 0, 0, 0, 0, loc), "2026-08-21"},
		{"utc midnight is 08:00 cst", time.Date(2026, 8, 20, 16, 0, 0, 0, time.UTC), "2026-08-21"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := taskstats.DateOf(tc.in, loc); got != tc.want {
				t.Fatalf("DateOf(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
