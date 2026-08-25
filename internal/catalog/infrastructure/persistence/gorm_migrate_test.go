package persistence_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"gorm.io/gorm"
)

func openMigrateTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := db.Open(db.Options{DSN: "file:legacy_" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := gdb.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return gdb
}

func TestRepairLegacyCaseIDRebuildsTextPrimaryKey(t *testing.T) {
	gdb := openMigrateTestDB(t)
	ctx := context.Background()

	if err := gdb.Exec(`CREATE TABLE catalog_cases (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		tags_json TEXT,
		cats_json TEXT,
		doc_json TEXT NOT NULL,
		enabled NUMERIC NOT NULL DEFAULT true,
		created_at DATETIME,
		updated_at DATETIME
	)`).Error; err != nil {
		t.Fatal(err)
	}
	docJSON, err := json.Marshal(map[string]any{
		"id": 1, "name": "Legacy Workflow",
		"inputs": []any{}, "outputs": []any{},
		"bindings": map[string]any{"workflow": map[string]any{}},
		"input_schema": map[string]any{"type": "object"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := gdb.Exec(
		`INSERT INTO catalog_cases (id, name, tags_json, cats_json, doc_json, enabled, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"1", "Legacy Workflow", "[]", "[]", string(docJSON), true, nil, nil,
	).Error; err != nil {
		t.Fatal(err)
	}
	// NULL-id leftovers from failed id=0 creates must be dropped by the repair.
	if err := gdb.Exec(
		`INSERT INTO catalog_cases (id, name, doc_json, enabled) VALUES (NULL, '未命名工作流', '{"id":0}', true)`,
	).Error; err != nil {
		t.Fatal(err)
	}

	if err := persistence.RepairLegacyCaseID(ctx, gdb); err != nil {
		t.Fatalf("repair: %v", err)
	}

	var idType string
	if err := gdb.Raw(`SELECT "type" FROM pragma_table_info('catalog_cases') WHERE name = 'id'`).Scan(&idType).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToLower(idType), "int") {
		t.Fatalf("id column still %q, want integer", idType)
	}
	var count int64
	if err := gdb.Model(&persistence.CaseRow{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("want 1 row after repair, got %d", count)
	}

	repo := persistence.NewGormRepository(gdb)
	got, err := repo.Get(ctx, 1)
	if err != nil {
		t.Fatalf("get migrated row: %v", err)
	}
	if got.Document.Name != "Legacy Workflow" {
		t.Fatalf("unexpected doc: %+v", got.Document)
	}
}

func TestRepairLegacyCaseIDNoopOnCurrentSchema(t *testing.T) {
	gdb := openMigrateTestDB(t)
	ctx := context.Background()
	if err := gdb.Migrator().AutoMigrate(&persistence.CaseRow{}); err != nil {
		t.Fatal(err)
	}
	if err := persistence.RepairLegacyCaseID(ctx, gdb); err != nil {
		t.Fatalf("repair: %v", err)
	}
	if gdb.Migrator().HasTable("catalog_cases_legacy") {
		t.Fatal("legacy table should not exist")
	}
}
