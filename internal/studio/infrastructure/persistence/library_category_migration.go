package persistence

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func MigrateLibraryCategories(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if tx.Migrator().HasTable("studio_library_folders") {
			var categories []LibraryCategoryRow
			if err := tx.Table("studio_library_folders").Find(&categories).Error; err != nil {
				return err
			}
			if len(categories) > 0 {
				if err := tx.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(&categories, 100).Error; err != nil {
					return err
				}
			}
		}
		if !tx.Migrator().HasColumn("studio_library_assets", "folder_id") {
			return nil
		}
		if err := tx.Table("studio_library_assets").
			Where("(category_id = '' OR category_id IS NULL) AND folder_id <> '' AND folder_id IS NOT NULL").
			Update("category_id", gorm.Expr("folder_id")).Error; err != nil {
			return err
		}
		return tx.Table("studio_library_assets").Where("folder_id <> '' AND folder_id IS NOT NULL").Update("folder_id", "").Error
	})
}
