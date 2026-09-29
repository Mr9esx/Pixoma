package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type AssetVersionPaletteRow struct {
	AssetVersionID   string     `gorm:"primaryKey;size:64"`
	AccountID        string     `gorm:"size:64;not null;index"`
	Status           string     `gorm:"size:24;not null;index:idx_studio_palette_queue,priority:1"`
	ColorsJSON       []byte     `gorm:"type:blob"`
	SamplePointsJSON []byte     `gorm:"type:blob"`
	AlgorithmVersion int        `gorm:"not null;default:1"`
	Attempts         int        `gorm:"not null;default:0"`
	LeaseUntil       *time.Time `gorm:"index:idx_studio_palette_queue,priority:3"`
	NextRetryAt      *time.Time `gorm:"index:idx_studio_palette_queue,priority:2"`
	AnalyzedAt       *time.Time
	ErrorCode        string `gorm:"size:64"`
	UpdatedAt        time.Time
}

func (AssetVersionPaletteRow) TableName() string { return "studio_asset_version_palettes" }

func newPendingPaletteRow(version domain.AssetVersion) *AssetVersionPaletteRow {
	return &AssetVersionPaletteRow{AssetVersionID: version.ID, AccountID: version.AccountID, Status: "pending", AlgorithmVersion: 1, UpdatedAt: version.CreatedAt}
}

func supportsPalette(mimeType string) bool {
	switch mimeType {
	case "image/png", "image/jpeg", "image/gif", "image/webp", "image/svg+xml", "video/mp4", "video/webm":
		return true
	default:
		return false
	}
}

func paletteFromRow(row AssetVersionPaletteRow) (*domain.AssetVersionPalette, error) {
	palette := &domain.AssetVersionPalette{AssetVersionID: row.AssetVersionID, AccountID: row.AccountID, Status: row.Status, AlgorithmVersion: row.AlgorithmVersion, Attempts: row.Attempts, LeaseUntil: row.LeaseUntil, NextRetryAt: row.NextRetryAt, AnalyzedAt: row.AnalyzedAt, ErrorCode: row.ErrorCode, UpdatedAt: row.UpdatedAt}
	if len(row.ColorsJSON) > 0 {
		if err := json.Unmarshal(row.ColorsJSON, &palette.Colors); err != nil {
			return nil, err
		}
	}
	if len(row.SamplePointsJSON) > 0 {
		if err := json.Unmarshal(row.SamplePointsJSON, &palette.SamplePoints); err != nil {
			return nil, err
		}
	}
	return palette, nil
}

func (r *GormRepository) GetAssetVersionPalette(ctx context.Context, accountID, versionID string) (*domain.AssetVersionPalette, error) {
	var row AssetVersionPaletteRow
	if err := r.db.WithContext(ctx).Where("account_id = ? AND asset_version_id = ?", accountID, versionID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return paletteFromRow(row)
}

func (r *GormRepository) ClaimPendingPaletteJobs(ctx context.Context, limit int, now time.Time) ([]*domain.AssetVersionPalette, error) {
	limit = normalizeLimit(limit)
	now = now.UTC()
	var candidates []AssetVersionPaletteRow
	if err := r.db.WithContext(ctx).Where("status = 'pending' OR (status = 'failed' AND attempts < 5 AND next_retry_at <= ?) OR (status = 'running' AND lease_until <= ?)", now, now).
		Order("updated_at ASC, asset_version_id ASC").Limit(limit).Find(&candidates).Error; err != nil {
		return nil, err
	}
	claimed := make([]*domain.AssetVersionPalette, 0, len(candidates))
	lease := now.Add(time.Minute)
	for _, row := range candidates {
		result := r.db.WithContext(ctx).Model(&AssetVersionPaletteRow{}).
			Where("asset_version_id = ? AND (status = 'pending' OR (status = 'failed' AND attempts < 5 AND next_retry_at <= ?) OR (status = 'running' AND lease_until <= ?))", row.AssetVersionID, now, now).
			Updates(map[string]any{"status": "running", "lease_until": lease, "attempts": gorm.Expr("attempts + 1"), "updated_at": now})
		if result.Error != nil {
			return nil, result.Error
		}
		if result.RowsAffected == 0 {
			continue
		}
		row.Status = "running"
		row.Attempts++
		row.LeaseUntil = &lease
		row.UpdatedAt = now
		palette, err := paletteFromRow(row)
		if err != nil {
			return nil, err
		}
		claimed = append(claimed, palette)
	}
	return claimed, nil
}

func (r *GormRepository) CompleteAssetVersionPalette(ctx context.Context, accountID, versionID string, colors []domain.PaletteColor, samplePoints []float64, widthPx, heightPx int, now time.Time) error {
	if len(colors) == 0 || len(colors) > 10 || widthPx < 0 || heightPx < 0 || (widthPx == 0) != (heightPx == 0) {
		return fmt.Errorf("%w: invalid palette colors", domain.ErrInvalid)
	}
	colorJSON, err := json.Marshal(colors)
	if err != nil {
		return err
	}
	sampleJSON, err := json.Marshal(samplePoints)
	if err != nil {
		return err
	}
	now = now.UTC()
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&AssetVersionPaletteRow{}).
			Where("account_id = ? AND asset_version_id = ? AND status = 'running'", accountID, versionID).
			Updates(map[string]any{"status": "ready", "colors_json": colorJSON, "sample_points_json": sampleJSON, "lease_until": nil, "next_retry_at": nil, "analyzed_at": now, "error_code": "", "updated_at": now})
		if err := resultError(result); err != nil {
			return err
		}
		if widthPx == 0 {
			return nil
		}
		return resultError(tx.Model(&AssetVersionRow{}).
			Where("account_id = ? AND id = ?", accountID, versionID).
			Updates(map[string]any{"width_px": widthPx, "height_px": heightPx}))
	})
}

func (r *GormRepository) FailAssetVersionPalette(ctx context.Context, accountID, versionID, errorCode string, nextRetryAt time.Time) error {
	result := r.db.WithContext(ctx).Model(&AssetVersionPaletteRow{}).
		Where("account_id = ? AND asset_version_id = ? AND status = 'running'", accountID, versionID).
		Updates(map[string]any{"status": "failed", "error_code": errorCode, "lease_until": nil, "next_retry_at": nextRetryAt.UTC(), "updated_at": time.Now().UTC()})
	return resultError(result)
}

func (r *GormRepository) RetryAssetVersionPalette(ctx context.Context, accountID, versionID string, now time.Time) error {
	result := r.db.WithContext(ctx).Model(&AssetVersionPaletteRow{}).
		Where("account_id = ? AND asset_version_id = ?", accountID, versionID).
		Updates(map[string]any{"status": "pending", "attempts": 0, "colors_json": nil, "sample_points_json": nil, "lease_until": nil, "next_retry_at": nil, "analyzed_at": nil, "error_code": "", "updated_at": now.UTC()})
	return resultError(result)
}
