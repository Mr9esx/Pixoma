package db

import (
	"fmt"

	"gorm.io/gorm"
)

// RenameLegacy moves pre-edge table/column names onto the current schema.
// Safe to run on a fresh database (no-op when names already match).
func RenameLegacy(gdb *gorm.DB) error {
	if gdb == nil {
		return fmt.Errorf("db: nil gdb")
	}
	m := gdb.Migrator()
	if m.HasTable("comfy_instances") && !m.HasTable("edges") {
		if err := m.RenameTable("comfy_instances", "edges"); err != nil {
			return fmt.Errorf("db: rename comfy_instances: %w", err)
		}
	}
	if m.HasTable("tasks") && m.HasColumn("tasks", "instance_id") && !m.HasColumn("tasks", "edge_id") {
		if err := gdb.Exec(`ALTER TABLE tasks RENAME COLUMN instance_id TO edge_id`).Error; err != nil {
			return fmt.Errorf("db: rename tasks.instance_id: %w", err)
		}
	}
	if m.HasTable("edges") && !m.HasColumn("edges", "hardware_refresh_requested") {
		if err := gdb.Exec(`ALTER TABLE edges ADD COLUMN hardware_refresh_requested INTEGER NOT NULL DEFAULT 0`).Error; err != nil {
			return fmt.Errorf("db: add edges.hardware_refresh_requested: %w", err)
		}
	}
	if m.HasTable("edges") && m.HasColumn("edges", "base_url") {
		if err := gdb.Exec(`ALTER TABLE edges DROP COLUMN base_url`).Error; err != nil {
			return fmt.Errorf("db: drop edges.base_url: %w", err)
		}
	}
	return nil
}
