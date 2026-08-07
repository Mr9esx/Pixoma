package db

import (
	"fmt"

	"gorm.io/gorm"
)

// AutoMigrate runs GORM AutoMigrate for the given models.
func AutoMigrate(gdb *gorm.DB, models ...any) error {
	if gdb == nil {
		return fmt.Errorf("db: nil gdb")
	}
	if err := gdb.AutoMigrate(models...); err != nil {
		return fmt.Errorf("db: automigrate: %w", err)
	}
	return nil
}
