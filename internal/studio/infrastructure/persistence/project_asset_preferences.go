package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type AssetLibraryPreferencesRow struct {
	AccountID     string `gorm:"primaryKey;size:64"`
	TreeMode      string `gorm:"size:24;not null"`
	LastProjectID string `gorm:"size:64;not null"`
	UpdatedAt     time.Time
}

func (AssetLibraryPreferencesRow) TableName() string { return "studio_asset_library_preferences" }

func validTreeMode(mode string) bool {
	switch mode {
	case "asset", "session", "category", "format", "rating", "tag":
		return true
	default:
		return false
	}
}

func (r *GormRepository) GetAssetLibraryPreferences(ctx context.Context, accountID string) (*domain.AssetLibraryPreferences, error) {
	if accountID == "" {
		return nil, fmt.Errorf("%w: account is required", domain.ErrInvalid)
	}
	var row AssetLibraryPreferencesRow
	if err := r.db.WithContext(ctx).Where("account_id = ?", accountID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &domain.AssetLibraryPreferences{AccountID: accountID, TreeMode: "asset"}, nil
		}
		return nil, err
	}
	return &domain.AssetLibraryPreferences{AccountID: row.AccountID, TreeMode: row.TreeMode, LastProjectID: row.LastProjectID, UpdatedAt: row.UpdatedAt}, nil
}

func (r *GormRepository) SaveAssetLibraryPreferences(ctx context.Context, preferences *domain.AssetLibraryPreferences) error {
	if preferences == nil || preferences.AccountID == "" || !validTreeMode(preferences.TreeMode) {
		return fmt.Errorf("%w: invalid preferences", domain.ErrInvalid)
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if preferences.LastProjectID != "" {
			var project ProjectRow
			if err := tx.Where("account_id = ? AND id = ?", preferences.AccountID, preferences.LastProjectID).First(&project).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return domain.ErrNotFound
				}
				return err
			}
		}
		row := &AssetLibraryPreferencesRow{AccountID: preferences.AccountID, TreeMode: preferences.TreeMode, LastProjectID: preferences.LastProjectID, UpdatedAt: preferences.UpdatedAt.UTC()}
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "account_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"tree_mode", "last_project_id", "updated_at"}),
		}).Create(row).Error
	})
}
