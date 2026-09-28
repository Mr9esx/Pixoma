package persistence

import (
	"context"

	"gorm.io/gorm"
)

func MigrateSessionProjectIDs(ctx context.Context, db *gorm.DB) error {
	if !db.Migrator().HasTable("studio_sessions") || !db.Migrator().HasColumn("studio_sessions", "project_id") {
		return nil
	}
	return db.WithContext(ctx).Table("studio_sessions").Where("project_id IS NULL").UpdateColumn("project_id", "").Error
}
