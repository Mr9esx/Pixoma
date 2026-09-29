package main

import (
	"testing"

	"github.com/Mr9esx/Pixoma/internal/platform/db"
)

func TestRebuildStudioDataKeepsModelConfiguration(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:rebuild_studio_data?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := gdb.Exec("CREATE TABLE studio_library_folders (id TEXT PRIMARY KEY)").Error; err != nil {
		t.Fatal(err)
	}
	if err := gdb.Exec("CREATE TABLE studio_model_configs (id TEXT PRIMARY KEY, name TEXT)").Error; err != nil {
		t.Fatal(err)
	}
	if err := gdb.Exec("INSERT INTO studio_model_configs (id, name) VALUES (?, ?)", "model-a", "保留的模型").Error; err != nil {
		t.Fatal(err)
	}
	if err := rebuildStudioData(gdb); err != nil {
		t.Fatal(err)
	}
	if gdb.Migrator().HasTable("studio_library_folders") {
		t.Fatal("legacy library folder table remains")
	}
	for _, table := range []string{"studio_assets", "studio_asset_versions", "studio_project_assets", "studio_session_asset_usages", "studio_asset_version_palettes"} {
		if !gdb.Migrator().HasTable(table) {
			t.Fatalf("missing table %s", table)
		}
	}
	var count int64
	if err := gdb.Table("studio_model_configs").Where("id = ? AND name = ?", "model-a", "保留的模型").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("model config count = %d", count)
	}
}
