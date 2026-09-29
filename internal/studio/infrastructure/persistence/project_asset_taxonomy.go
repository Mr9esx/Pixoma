package persistence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type AssetCategoryRow struct {
	ID        string `gorm:"primaryKey;size:64"`
	AccountID string `gorm:"size:64;not null;uniqueIndex:idx_studio_asset_category_name"`
	ProjectID string `gorm:"size:64;not null;uniqueIndex:idx_studio_asset_category_name;index"`
	ParentID  string `gorm:"size:64;not null;uniqueIndex:idx_studio_asset_category_name"`
	Name      string `gorm:"size:256;not null;uniqueIndex:idx_studio_asset_category_name"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (AssetCategoryRow) TableName() string { return "studio_asset_categories" }

type AssetTagRow struct {
	ID        string `gorm:"primaryKey;size:64"`
	AccountID string `gorm:"size:64;not null;uniqueIndex:idx_studio_asset_tag_name"`
	Name      string `gorm:"size:256;not null;uniqueIndex:idx_studio_asset_tag_name"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (AssetTagRow) TableName() string { return "studio_asset_tags" }

type ProjectAssetTagRow struct {
	ProjectAssetID string `gorm:"primaryKey;size:64"`
	TagID          string `gorm:"primaryKey;size:64;index:idx_studio_project_asset_tag_lookup,priority:2"`
	AccountID      string `gorm:"size:64;not null;index:idx_studio_project_asset_tag_lookup,priority:1"`
}

func (ProjectAssetTagRow) TableName() string { return "studio_project_asset_tags" }

func (r *GormRepository) CreateAssetCategory(ctx context.Context, category *domain.AssetCategory) error {
	if category == nil || category.ID == "" || category.AccountID == "" || strings.TrimSpace(category.Name) == "" {
		return fmt.Errorf("%w: invalid category", domain.ErrInvalid)
	}
	category.Name = strings.TrimSpace(category.Name)
	return translateCreateError(r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if category.ProjectID != "" {
			var project ProjectRow
			if err := tx.Where("account_id = ? AND id = ?", category.AccountID, category.ProjectID).First(&project).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return domain.ErrNotFound
				}
				return err
			}
		}
		if category.ParentID != "" {
			var parent AssetCategoryRow
			if err := tx.Where("account_id = ? AND project_id = ? AND id = ?", category.AccountID, category.ProjectID, category.ParentID).First(&parent).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return domain.ErrNotFound
				}
				return err
			}
		}
		return tx.Create(&AssetCategoryRow{ID: category.ID, AccountID: category.AccountID, ProjectID: category.ProjectID, ParentID: category.ParentID, Name: category.Name, CreatedAt: category.CreatedAt, UpdatedAt: category.UpdatedAt}).Error
	}))
}

func (r *GormRepository) GetAssetCategory(ctx context.Context, accountID, categoryID string) (*domain.AssetCategory, error) {
	var row AssetCategoryRow
	if err := r.db.WithContext(ctx).Where("account_id = ? AND id = ?", accountID, categoryID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &domain.AssetCategory{ID: row.ID, AccountID: row.AccountID, ProjectID: row.ProjectID, ParentID: row.ParentID, Name: row.Name, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

func (r *GormRepository) ListAssetCategories(ctx context.Context, accountID, projectID string) ([]*domain.AssetCategory, error) {
	var rows []AssetCategoryRow
	if err := r.db.WithContext(ctx).Where("account_id = ? AND project_id = ?", accountID, projectID).Order("name ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.AssetCategory, 0, len(rows))
	for _, row := range rows {
		out = append(out, &domain.AssetCategory{ID: row.ID, AccountID: row.AccountID, ProjectID: row.ProjectID, ParentID: row.ParentID, Name: row.Name, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt})
	}
	return out, nil
}

func (r *GormRepository) UpdateAssetCategory(ctx context.Context, accountID, categoryID, parentID, name string, now time.Time) error {
	name = strings.TrimSpace(name)
	if accountID == "" || categoryID == "" || name == "" || len([]rune(name)) > 128 || categoryID == parentID {
		return fmt.Errorf("%w: invalid category update", domain.ErrInvalid)
	}
	return translateCreateError(r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var category AssetCategoryRow
		if err := tx.Where("account_id = ? AND id = ?", accountID, categoryID).First(&category).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return err
		}
		if parentID != "" {
			var rows []AssetCategoryRow
			if err := tx.Where("account_id = ? AND project_id = ?", accountID, category.ProjectID).Find(&rows).Error; err != nil {
				return err
			}
			parents := make(map[string]string, len(rows))
			for _, row := range rows {
				parents[row.ID] = row.ParentID
			}
			if _, ok := parents[parentID]; !ok {
				return domain.ErrNotFound
			}
			for current := parentID; current != ""; current = parents[current] {
				if current == categoryID {
					return fmt.Errorf("%w: category cycle", domain.ErrInvalid)
				}
			}
		}
		return tx.Model(&AssetCategoryRow{}).Where("account_id = ? AND id = ?", accountID, categoryID).
			Updates(map[string]any{"parent_id": parentID, "name": name, "updated_at": now.UTC()}).Error
	}))
}

func (r *GormRepository) DeleteAssetCategory(ctx context.Context, accountID, categoryID string) error {
	return translateCreateError(r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var category AssetCategoryRow
		if err := tx.Where("account_id = ? AND id = ?", accountID, categoryID).First(&category).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return err
		}
		if err := tx.Model(&AssetCategoryRow{}).Where("account_id = ? AND project_id = ? AND parent_id = ?", accountID, category.ProjectID, categoryID).
			Update("parent_id", category.ParentID).Error; err != nil {
			return err
		}
		if err := tx.Model(&ProjectAssetRow{}).Where("account_id = ? AND project_id = ? AND category_id = ?", accountID, category.ProjectID, categoryID).
			Update("category_id", category.ParentID).Error; err != nil {
			return err
		}
		return resultError(tx.Where("account_id = ? AND id = ?", accountID, categoryID).Delete(&AssetCategoryRow{}))
	}))
}

func (r *GormRepository) CreateAssetTag(ctx context.Context, tag *domain.AssetTag) error {
	if tag == nil || tag.ID == "" || tag.AccountID == "" || strings.TrimSpace(tag.Name) == "" {
		return fmt.Errorf("%w: invalid tag", domain.ErrInvalid)
	}
	tag.Name = strings.TrimSpace(tag.Name)
	return translateCreateError(r.db.WithContext(ctx).Create(&AssetTagRow{ID: tag.ID, AccountID: tag.AccountID, Name: tag.Name, CreatedAt: tag.CreatedAt, UpdatedAt: tag.UpdatedAt}).Error)
}

func (r *GormRepository) ListAssetTags(ctx context.Context, accountID string) ([]*domain.AssetTag, error) {
	var rows []AssetTagRow
	if err := r.db.WithContext(ctx).Where("account_id = ?", accountID).Order("name ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.AssetTag, 0, len(rows))
	for _, row := range rows {
		out = append(out, &domain.AssetTag{ID: row.ID, AccountID: row.AccountID, Name: row.Name, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt})
	}
	return out, nil
}

func (r *GormRepository) RenameAssetTag(ctx context.Context, accountID, tagID, name string, now time.Time) error {
	name = strings.TrimSpace(name)
	if accountID == "" || tagID == "" || name == "" || len([]rune(name)) > 128 {
		return fmt.Errorf("%w: invalid tag name", domain.ErrInvalid)
	}
	result := r.db.WithContext(ctx).Model(&AssetTagRow{}).Where("account_id = ? AND id = ?", accountID, tagID).
		Updates(map[string]any{"name": name, "updated_at": now.UTC()})
	if result.Error != nil {
		return translateCreateError(result.Error)
	}
	return resultError(result)
}

func (r *GormRepository) SetProjectAssetTags(ctx context.Context, accountID, projectAssetID string, tagIDs []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var entry ProjectAssetRow
		if err := tx.Where("account_id = ? AND id = ?", accountID, projectAssetID).First(&entry).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return err
		}
		seen := make(map[string]bool, len(tagIDs))
		rows := make([]ProjectAssetTagRow, 0, len(tagIDs))
		for _, id := range tagIDs {
			if id == "" {
				return fmt.Errorf("%w: empty tag", domain.ErrInvalid)
			}
			if seen[id] {
				continue
			}
			seen[id] = true
			var tag AssetTagRow
			if err := tx.Where("account_id = ? AND id = ?", accountID, id).First(&tag).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return domain.ErrNotFound
				}
				return err
			}
			rows = append(rows, ProjectAssetTagRow{ProjectAssetID: projectAssetID, TagID: id, AccountID: accountID})
		}
		if err := tx.Where("account_id = ? AND project_asset_id = ?", accountID, projectAssetID).Delete(&ProjectAssetTagRow{}).Error; err != nil {
			return err
		}
		if len(rows) > 0 {
			return tx.Create(&rows).Error
		}
		return nil
	})
}
