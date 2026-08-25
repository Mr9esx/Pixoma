package persistence

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// RepairLegacyCaseID rebuilds catalog_cases when the table still uses the
// pre-refactor TEXT primary key (SQLite). AutoMigrate does not alter column
// types, so legacy tables would store id=NULL on create and break create→get.
// Rows without a usable numeric id (including NULL-id leftovers) are dropped.
// No-op on fresh databases and non-SQLite drivers.
func RepairLegacyCaseID(ctx context.Context, gdb *gorm.DB) error {
	if gdb == nil {
		return fmt.Errorf("db: nil gdb")
	}
	if gdb.Dialector.Name() != "sqlite" {
		return nil
	}
	if !gdb.Migrator().HasTable("catalog_cases") {
		return nil
	}
	var idType string
	if err := gdb.WithContext(ctx).Raw(
		`SELECT "type" FROM pragma_table_info('catalog_cases') WHERE name = 'id'`,
	).Scan(&idType).Error; err != nil {
		return fmt.Errorf("db: inspect catalog_cases.id: %w", err)
	}
	if strings.Contains(strings.ToLower(idType), "int") {
		return nil
	}
	if err := gdb.Migrator().RenameTable("catalog_cases", "catalog_cases_legacy"); err != nil {
		return fmt.Errorf("db: rename catalog_cases: %w", err)
	}
	if err := gdb.Migrator().AutoMigrate(&CaseRow{}); err != nil {
		return fmt.Errorf("db: recreate catalog_cases: %w", err)
	}
	if err := gdb.WithContext(ctx).Exec(`
		INSERT INTO catalog_cases (id, name, tags_json, cats_json, doc_json, enabled, created_at, updated_at)
		SELECT CAST(id AS INTEGER), name, tags_json, cats_json, doc_json, enabled, created_at, updated_at
		FROM catalog_cases_legacy
		WHERE id IS NOT NULL AND TRIM(id) <> '' AND CAST(id AS INTEGER) > 0
	`).Error; err != nil {
		return fmt.Errorf("db: copy catalog_cases rows: %w", err)
	}
	if err := gdb.Migrator().DropTable("catalog_cases_legacy"); err != nil {
		return fmt.Errorf("db: drop catalog_cases_legacy: %w", err)
	}
	return nil
}
