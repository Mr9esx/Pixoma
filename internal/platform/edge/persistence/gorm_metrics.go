package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// MetricsRow is the GORM model for the edge_metrics table.
type MetricsRow struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement"`
	EdgeID      string    `gorm:"size:128;index"`
	MetricsJSON string    `gorm:"type:text"`
	CollectedAt time.Time `gorm:"index"`
}

func (MetricsRow) TableName() string { return "edge_metrics" }

// MetricsRepository is a GORM-backed edge.MetricsRepository.
type MetricsRepository struct {
	db        *gorm.DB
	retention time.Duration
}

func NewMetricsRepository(gdb *gorm.DB, retention time.Duration) *MetricsRepository {
	return &MetricsRepository{db: gdb, retention: retention}
}

func (r *MetricsRepository) Append(ctx context.Context, edgeID sharedkernel.EdgeID, m edge.Metrics) error {
	if edgeID == "" || edge.MetricsEmpty(m) {
		return fmt.Errorf("edge: invalid metrics append")
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("edge: metrics_json: %w", err)
	}
	row := MetricsRow{EdgeID: string(edgeID), MetricsJSON: string(raw), CollectedAt: m.CollectedAt}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	cutoff := time.Now().UTC().Add(-r.retention)
	return r.db.WithContext(ctx).Where("collected_at < ?", cutoff).Delete(&MetricsRow{}).Error
}

func (r *MetricsRepository) ListSince(ctx context.Context, edgeID sharedkernel.EdgeID, since time.Time, limit int) ([]edge.Metrics, error) {
	if limit <= 0 {
		limit = 720
	}
	var rows []MetricsRow
	if err := r.db.WithContext(ctx).
		Where("edge_id = ? AND collected_at >= ?", string(edgeID), since).
		Order("collected_at ASC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]edge.Metrics, 0, len(rows))
	for _, row := range rows {
		var m edge.Metrics
		if err := json.Unmarshal([]byte(row.MetricsJSON), &m); err != nil {
			return nil, fmt.Errorf("edge: metrics_json: %w", err)
		}
		out = append(out, m)
	}
	return out, nil
}

// LatestAll returns the most recent snapshot per edge collected at/after since.
func (r *MetricsRepository) LatestAll(ctx context.Context, since time.Time) (map[sharedkernel.EdgeID]edge.Metrics, error) {
	sub := r.db.WithContext(ctx).
		Model(&MetricsRow{}).
		Select("MAX(id)").
		Where("collected_at >= ?", since).
		Group("edge_id")
	var rows []MetricsRow
	if err := r.db.WithContext(ctx).Where("id IN (?)", sub).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[sharedkernel.EdgeID]edge.Metrics, len(rows))
	for _, row := range rows {
		var m edge.Metrics
		if err := json.Unmarshal([]byte(row.MetricsJSON), &m); err != nil {
			return nil, fmt.Errorf("edge: metrics_json: %w", err)
		}
		out[sharedkernel.EdgeID(row.EdgeID)] = m
	}
	return out, nil
}

var _ edge.MetricsRepository = (*MetricsRepository)(nil)
