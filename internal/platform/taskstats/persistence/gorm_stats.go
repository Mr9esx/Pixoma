package persistence

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/taskstats"
)

// DailyStatsRow is the GORM model for the task_daily_stats table.
type DailyStatsRow struct {
	StatDate        string    `gorm:"column:stat_date;primaryKey;size:10"`
	ProcessedCount  int       `gorm:"column:processed_count;not null;default:0"`
	SucceededCount  int       `gorm:"column:succeeded_count;not null;default:0"`
	FailedCount     int       `gorm:"column:failed_count;not null;default:0"`
	CancelledCount  int       `gorm:"column:cancelled_count;not null;default:0"`
	TotalDurationMS int64     `gorm:"column:total_duration_ms;not null;default:0"`
	UpdatedAt       time.Time `gorm:"column:updated_at;not null"`
}

func (DailyStatsRow) TableName() string { return "task_daily_stats" }

// EdgeDailyStatsRow is the GORM model for the task_edge_daily_stats table.
type EdgeDailyStatsRow struct {
	StatDate       string    `gorm:"column:stat_date;primaryKey;size:10"`
	EdgeID         string    `gorm:"column:edge_id;primaryKey;size:64"`
	ProcessedCount int       `gorm:"column:processed_count;not null;default:0"`
	UpdatedAt      time.Time `gorm:"column:updated_at;not null"`
}

func (EdgeDailyStatsRow) TableName() string { return "task_edge_daily_stats" }

// ErrorDailyStatsRow is the GORM model for the task_error_daily_stats table.
type ErrorDailyStatsRow struct {
	StatDate  string    `gorm:"column:stat_date;primaryKey;size:10"`
	ErrorCode string    `gorm:"column:error_code;primaryKey;size:128"`
	Count     int       `gorm:"column:count;not null;default:0"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null"`
}

func (ErrorDailyStatsRow) TableName() string { return "task_error_daily_stats" }

// GormStatsRepository is a GORM-backed taskstats.Repository.
type GormStatsRepository struct {
	db        *gorm.DB
	retention time.Duration
	loc       *time.Location
}

func NewGormStatsRepository(gdb *gorm.DB, retention time.Duration, loc *time.Location) *GormStatsRepository {
	return &GormStatsRepository{db: gdb, retention: retention, loc: loc}
}

func (r *GormStatsRepository) AddTerminal(ctx context.Context, in taskstats.AddTerminalInput) error {
	date := taskstats.DateOf(in.CompletedAt, r.loc)
	now := time.Now().UTC()
	dur := in.CompletedAt.Sub(in.CreatedAt).Milliseconds()
	if dur < 0 {
		dur = 0
	}
	daily := DailyStatsRow{StatDate: date, ProcessedCount: 1, UpdatedAt: now, TotalDurationMS: dur}
	switch in.Status {
	case taskstats.StatusSucceeded:
		daily.SucceededCount = 1
	case taskstats.StatusFailed:
		daily.FailedCount = 1
	case taskstats.StatusCancelled:
		daily.CancelledCount = 1
	default:
		return fmt.Errorf("taskstats: unsupported terminal status %q", in.Status)
	}
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "stat_date"}},
		DoUpdates: clause.Assignments(map[string]any{
			"processed_count":   gorm.Expr("processed_count + 1"),
			"succeeded_count":   gorm.Expr("succeeded_count + ?", daily.SucceededCount),
			"failed_count":      gorm.Expr("failed_count + ?", daily.FailedCount),
			"cancelled_count":   gorm.Expr("cancelled_count + ?", daily.CancelledCount),
			"total_duration_ms": gorm.Expr("total_duration_ms + ?", daily.TotalDurationMS),
			"updated_at":        now,
		}),
	}).Create(&daily).Error; err != nil {
		return err
	}
	if in.EdgeID != "" {
		edgeRow := EdgeDailyStatsRow{StatDate: date, EdgeID: in.EdgeID, ProcessedCount: 1, UpdatedAt: now}
		if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "stat_date"}, {Name: "edge_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"processed_count": gorm.Expr("processed_count + 1"),
				"updated_at":      now,
			}),
		}).Create(&edgeRow).Error; err != nil {
			return err
		}
	}
	if in.ErrorCode != "" {
		errRow := ErrorDailyStatsRow{StatDate: date, ErrorCode: in.ErrorCode, Count: 1, UpdatedAt: now}
		if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "stat_date"}, {Name: "error_code"}},
			DoUpdates: clause.Assignments(map[string]any{
				"count":      gorm.Expr("count + 1"),
				"updated_at": now,
			}),
		}).Create(&errRow).Error; err != nil {
			return err
		}
	}
	return r.Prune(ctx, taskstats.DateOf(time.Now().In(r.loc).Add(-r.retention), r.loc))
}

func (r *GormStatsRepository) ListDaily(ctx context.Context, from, to string) ([]taskstats.DailyRow, error) {
	var rows []DailyStatsRow
	if err := r.db.WithContext(ctx).
		Where("stat_date BETWEEN ? AND ?", from, to).
		Order("stat_date ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]taskstats.DailyRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, taskstats.DailyRow{
			Date:            row.StatDate,
			Processed:       row.ProcessedCount,
			Succeeded:       row.SucceededCount,
			Failed:          row.FailedCount,
			Cancelled:       row.CancelledCount,
			TotalDurationMS: row.TotalDurationMS,
		})
	}
	return out, nil
}

func (r *GormStatsRepository) ListErrors(ctx context.Context, from, to string, limit int) ([]taskstats.ErrorRow, error) {
	type item struct {
		ErrorCode string
		Total     int
	}
	var items []item
	if err := r.db.WithContext(ctx).Model(&ErrorDailyStatsRow{}).
		Select("error_code", "SUM(count) AS total").
		Where("stat_date BETWEEN ? AND ?", from, to).
		Group("error_code").Order("total DESC").Limit(limit).
		Scan(&items).Error; err != nil {
		return nil, err
	}
	out := make([]taskstats.ErrorRow, 0, len(items))
	for _, it := range items {
		out = append(out, taskstats.ErrorRow{ErrorCode: it.ErrorCode, Count: it.Total})
	}
	return out, nil
}

func (r *GormStatsRepository) ListEdges(ctx context.Context, from, to string) ([]taskstats.EdgeRow, error) {
	type item struct {
		EdgeID string
		Total  int
	}
	var items []item
	if err := r.db.WithContext(ctx).Model(&EdgeDailyStatsRow{}).
		Select("edge_id", "SUM(processed_count) AS total").
		Where("stat_date BETWEEN ? AND ?", from, to).
		Group("edge_id").Order("total DESC").
		Scan(&items).Error; err != nil {
		return nil, err
	}
	out := make([]taskstats.EdgeRow, 0, len(items))
	for _, it := range items {
		out = append(out, taskstats.EdgeRow{EdgeID: it.EdgeID, Count: it.Total})
	}
	return out, nil
}

func (r *GormStatsRepository) Prune(ctx context.Context, before string) error {
	for _, m := range []any{&DailyStatsRow{}, &EdgeDailyStatsRow{}, &ErrorDailyStatsRow{}} {
		if err := r.db.WithContext(ctx).Where("stat_date < ?", before).Delete(m).Error; err != nil {
			return err
		}
	}
	return nil
}

var _ taskstats.Repository = (*GormStatsRepository)(nil)
