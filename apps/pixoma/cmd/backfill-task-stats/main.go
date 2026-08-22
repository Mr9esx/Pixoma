package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm/clause"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/appboot"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/taskstats"
	taskstatspersist "github.com/mr9esx/comfyui_tgbot/internal/platform/taskstats/persistence"
	taskpersist "github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/persistence"
)

// backfill-task-stats recomputes the task stats tables from the tasks table
// (authoritative source) as absolute per-day values. It is idempotent and safe
// to re-run. Config comes from env: DB_DRIVER (default sqlite), DATABASE_DSN
// (default DATA_DIR/app.db), DATA_DIR (default data).
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	driver, dsn := resolveBackfillDSN()
	ctx := context.Background()
	gdb, cleanup, err := appboot.Bootstrap(ctx, appboot.Options{
		Driver: driver,
		DSN:    dsn,
		Models: []any{
			&taskpersist.TaskRow{},
			&taskstatspersist.DailyStatsRow{},
			&taskstatspersist.EdgeDailyStatsRow{},
			&taskstatspersist.ErrorDailyStatsRow{},
			&taskstatspersist.CaseDailyStatsRow{},
		},
	})
	if err != nil {
		slog.Error("open db", "err", err)
		os.Exit(1)
	}
	defer func() { _ = cleanup() }()

	loc := statsLocation()
	now := time.Now().UTC()
	const batchSize = 1000
	type dayAgg struct {
		processed, succ, fail, canc int
		dur, queue, exec            int64
	}
	type caseAgg struct {
		count int
		dur   int64
	}
	byDate := map[string]*dayAgg{}
	byEdge := map[string]map[string]int{}
	byEdgeSucc := map[string]map[string]int{}
	byEdgeFail := map[string]map[string]int{}
	byErr := map[string]map[string]int{}
	byCase := map[string]map[uint64]*caseAgg{}

	for offset := 0; ; offset += batchSize {
		var rows []taskpersist.TaskRow
		if err := gdb.
			Where("status IN ?", []string{"succeeded", "failed", "cancelled"}).
			Order("id ASC").
			Limit(batchSize).Offset(offset).
			Find(&rows).Error; err != nil {
			slog.Error("query tasks", "err", err)
			os.Exit(1)
		}
		if len(rows) == 0 {
			break
		}
		for _, row := range rows {
			completedAt := row.CompletedAt
			if completedAt.IsZero() {
				completedAt = row.UpdatedAt
			}
			d := taskstats.DateOf(completedAt, loc)
			a := byDate[d]
			if a == nil {
				a = &dayAgg{}
				byDate[d] = a
			}
			a.processed++
			dur := completedAt.Sub(row.CreatedAt).Milliseconds()
			if dur < 0 {
				dur = 0
			}
			a.dur += dur
			queue := row.StartedAt.Sub(row.CreatedAt).Milliseconds()
			if queue < 0 {
				queue = 0
			}
			exec := completedAt.Sub(row.StartedAt).Milliseconds()
			if exec < 0 {
				exec = 0
			}
			a.queue += queue
			a.exec += exec
			switch row.Status {
			case "succeeded":
				a.succ++
				if byEdgeSucc[d] == nil {
					byEdgeSucc[d] = map[string]int{}
				}
				if row.EdgeID != "" {
					byEdgeSucc[d][row.EdgeID]++
				}
			case "failed":
				a.fail++
				if byEdgeFail[d] == nil {
					byEdgeFail[d] = map[string]int{}
				}
				if row.EdgeID != "" {
					byEdgeFail[d][row.EdgeID]++
				}
				if row.ErrorCode != "" {
					if byErr[d] == nil {
						byErr[d] = map[string]int{}
					}
					byErr[d][row.ErrorCode]++
				}
			case "cancelled":
				a.canc++
			}
			if row.EdgeID != "" {
				if byEdge[d] == nil {
					byEdge[d] = map[string]int{}
				}
				byEdge[d][row.EdgeID]++
			}
			if row.CaseID != 0 {
				if byCase[d] == nil {
					byCase[d] = map[uint64]*caseAgg{}
				}
				c := byCase[d][row.CaseID]
				if c == nil {
					c = &caseAgg{}
					byCase[d][row.CaseID] = c
				}
				c.count++
				c.dur += dur
			}
		}
		if len(rows) < batchSize {
			break
		}
	}

	for d, a := range byDate {
		row := taskstatspersist.DailyStatsRow{
			StatDate: d, ProcessedCount: a.processed, SucceededCount: a.succ,
			FailedCount: a.fail, CancelledCount: a.canc, TotalDurationMS: a.dur,
			TotalQueueMS: a.queue, TotalExecMS: a.exec, UpdatedAt: now,
		}
		if err := gdb.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "stat_date"}},
			DoUpdates: clause.Assignments(map[string]any{
				"processed_count":   a.processed,
				"succeeded_count":   a.succ,
				"failed_count":      a.fail,
				"cancelled_count":   a.canc,
				"total_duration_ms": a.dur,
				"total_queue_ms":    a.queue,
				"total_exec_ms":     a.exec,
				"updated_at":        now,
			}),
		}).Create(&row).Error; err != nil {
			slog.Error("upsert daily", "err", err)
			os.Exit(1)
		}
	}
	for d, edges := range byEdge {
		for edgeID, count := range edges {
			succ := byEdgeSucc[d][edgeID]
			fail := byEdgeFail[d][edgeID]
			row := taskstatspersist.EdgeDailyStatsRow{StatDate: d, EdgeID: edgeID, ProcessedCount: count, SucceededCount: succ, FailedCount: fail, UpdatedAt: now}
			if err := gdb.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "stat_date"}, {Name: "edge_id"}},
				DoUpdates: clause.Assignments(map[string]any{"processed_count": count, "succeeded_count": succ, "failed_count": fail, "updated_at": now}),
			}).Create(&row).Error; err != nil {
				slog.Error("upsert edge", "err", err)
				os.Exit(1)
			}
		}
	}
	for d, codes := range byErr {
		for code, count := range codes {
			row := taskstatspersist.ErrorDailyStatsRow{StatDate: d, ErrorCode: code, Count: count, UpdatedAt: now}
			if err := gdb.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "stat_date"}, {Name: "error_code"}},
				DoUpdates: clause.Assignments(map[string]any{"count": count, "updated_at": now}),
			}).Create(&row).Error; err != nil {
				slog.Error("upsert error", "err", err)
				os.Exit(1)
			}
		}
	}
	for d, cases := range byCase {
		for caseID, c := range cases {
			row := taskstatspersist.CaseDailyStatsRow{StatDate: d, CaseID: caseID, Count: c.count, TotalDurationMS: c.dur, UpdatedAt: now}
			if err := gdb.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "stat_date"}, {Name: "case_id"}},
				DoUpdates: clause.Assignments(map[string]any{"count": c.count, "total_duration_ms": c.dur, "updated_at": now}),
			}).Create(&row).Error; err != nil {
				slog.Error("upsert case", "err", err)
				os.Exit(1)
			}
		}
	}

	repo := taskstatspersist.NewGormStatsRepository(gdb, statsRetention(), loc)
	if err := repo.Prune(ctx, taskstats.DateOf(time.Now().In(loc).Add(-statsRetention()), loc)); err != nil {
		slog.Error("prune", "err", err)
		os.Exit(1)
	}
	slog.Info("backfill complete", "days", len(byDate))
}

func resolveBackfillDSN() (driver, dsn string) {
	driver = strings.TrimSpace(os.Getenv("DB_DRIVER"))
	if driver == "" {
		driver = "sqlite"
	}
	dsn = strings.TrimSpace(os.Getenv("DATABASE_DSN"))
	if dsn != "" {
		return driver, dsn
	}
	dataDir := strings.TrimSpace(os.Getenv("DATA_DIR"))
	if dataDir == "" {
		dataDir = "data"
	}
	return driver, filepath.Join(dataDir, "app.db")
}

func statsRetention() time.Duration {
	if v := strings.TrimSpace(os.Getenv("TASK_STATS_RETENTION")); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return 365 * 24 * time.Hour
}

func statsLocation() *time.Location {
	v := strings.TrimSpace(os.Getenv("STATS_TIMEZONE"))
	if v == "" {
		v = "Asia/Shanghai"
	}
	loc, err := time.LoadLocation(v)
	if err != nil {
		return time.FixedZone("Asia/Shanghai", 8*3600)
	}
	return loc
}
