package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm/clause"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/adminconfig"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/appboot"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/taskstats"
	taskstatspersist "github.com/mr9esx/comfyui_tgbot/internal/platform/taskstats/persistence"
	taskpersist "github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/persistence"
)

// backfill-task-stats recomputes the three task stats tables from the tasks
// table (authoritative source) as absolute per-day values. It is idempotent
// and safe to re-run.
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := adminconfig.Load("")
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}
	dsn := resolveDSN(cfg.DatabaseDSN)
	ctx := context.Background()
	gdb, cleanup, err := appboot.Bootstrap(ctx, appboot.Options{
		DSN: dsn,
		Models: []any{
			&taskpersist.TaskRow{},
			&taskstatspersist.DailyStatsRow{},
			&taskstatspersist.EdgeDailyStatsRow{},
			&taskstatspersist.ErrorDailyStatsRow{},
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
		dur                         int64
	}
	byDate := map[string]*dayAgg{}
	byEdge := map[string]map[string]int{}
	byErr := map[string]map[string]int{}

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
			d := taskstats.DateOf(row.CompletedAt, loc)
			a := byDate[d]
			if a == nil {
				a = &dayAgg{}
				byDate[d] = a
			}
			a.processed++
			dur := row.CompletedAt.Sub(row.CreatedAt).Milliseconds()
			if dur < 0 {
				dur = 0
			}
			a.dur += dur
			switch row.Status {
			case "succeeded":
				a.succ++
			case "failed":
				a.fail++
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
		}
		if len(rows) < batchSize {
			break
		}
	}

	for d, a := range byDate {
		row := taskstatspersist.DailyStatsRow{
			StatDate: d, ProcessedCount: a.processed, SucceededCount: a.succ,
			FailedCount: a.fail, CancelledCount: a.canc, TotalDurationMS: a.dur, UpdatedAt: now,
		}
		if err := gdb.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "stat_date"}},
			DoUpdates: clause.Assignments(map[string]any{
				"processed_count":   a.processed,
				"succeeded_count":   a.succ,
				"failed_count":      a.fail,
				"cancelled_count":   a.canc,
				"total_duration_ms": a.dur,
				"updated_at":        now,
			}),
		}).Create(&row).Error; err != nil {
			slog.Error("upsert daily", "err", err)
			os.Exit(1)
		}
	}
	for d, edges := range byEdge {
		for edgeID, count := range edges {
			row := taskstatspersist.EdgeDailyStatsRow{StatDate: d, EdgeID: edgeID, ProcessedCount: count, UpdatedAt: now}
			if err := gdb.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "stat_date"}, {Name: "edge_id"}},
				DoUpdates: clause.Assignments(map[string]any{"processed_count": count, "updated_at": now}),
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

	repo := taskstatspersist.NewGormStatsRepository(gdb, statsRetention(), loc)
	if err := repo.Prune(ctx, taskstats.DateOf(time.Now().In(loc).Add(-statsRetention()), loc)); err != nil {
		slog.Error("prune", "err", err)
		os.Exit(1)
	}
	slog.Info("backfill complete", "days", len(byDate))
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

func resolveDSN(configured string) string {
	if configured != "" {
		return configured
	}
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}
	return filepath.Join(dataDir, "app.db")
}
