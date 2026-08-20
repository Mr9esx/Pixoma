package db

import (
	"testing"

	"gorm.io/gorm"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge/persistence"
)

func TestRenameLegacy_ComfyInstancesAndTaskColumn(t *testing.T) {
	gdb := openRenameTestDB(t)
	if err := gdb.Exec(`CREATE TABLE comfy_instances (id TEXT PRIMARY KEY, base_url TEXT)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := gdb.Exec(`INSERT INTO comfy_instances (id, base_url) VALUES ('local', 'http://127.0.0.1:8188')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := gdb.Exec(`CREATE TABLE tasks (id TEXT PRIMARY KEY, instance_id TEXT)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := gdb.Exec(`INSERT INTO tasks (id, instance_id) VALUES ('t1', 'local')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := RenameLegacy(gdb); err != nil {
		t.Fatal(err)
	}
	if gdb.Migrator().HasTable("comfy_instances") {
		t.Fatal("old table still exists")
	}
	if !gdb.Migrator().HasTable("edges") {
		t.Fatal("edges missing")
	}
	if gdb.Migrator().HasColumn("tasks", "instance_id") {
		t.Fatal("old column still exists")
	}
	if !gdb.Migrator().HasColumn("tasks", "edge_id") {
		t.Fatal("edge_id missing")
	}
	if gdb.Migrator().HasColumn("edges", "base_url") {
		t.Fatal("base_url must be dropped")
	}
	var id string
	if err := gdb.Raw(`SELECT id FROM edges WHERE id = ?`, "local").Scan(&id).Error; err != nil || id != "local" {
		t.Fatalf("row migrated: id=%s err=%v", id, err)
	}
}

func TestRenameLegacy_ExistingEdgesAcceptHardwareRefreshColumn(t *testing.T) {
	gdb := openRenameTestDB(t)
	now := "2026-08-17T00:00:00Z"
	if err := gdb.Exec(`CREATE TABLE edges (
		id TEXT PRIMARY KEY,
		name TEXT,
		description TEXT,
		base_url TEXT NOT NULL,
		enabled INTEGER NOT NULL,
		capabilities_json TEXT NOT NULL,
		agent_token_enc TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := gdb.Exec(
		`INSERT INTO edges (id, name, description, base_url, enabled, capabilities_json, created_at, updated_at)
		 VALUES ('local', 'local', '', 'http://127.0.0.1:8188', 1, '[]', ?, ?)`,
		now, now,
	).Error; err != nil {
		t.Fatal(err)
	}
	if err := RenameLegacy(gdb); err != nil {
		t.Fatal(err)
	}
	if err := AutoMigrate(gdb, &persistence.EdgeRow{}); err != nil {
		t.Fatalf("automigrate existing edges: %v", err)
	}
	if gdb.Migrator().HasColumn("edges", "base_url") {
		t.Fatal("base_url must be dropped")
	}
	if !gdb.Migrator().HasColumn("edges", "hardware_refresh_requested") {
		t.Fatal("hardware_refresh_requested missing")
	}
	var requested bool
	if err := gdb.Raw(`SELECT hardware_refresh_requested FROM edges WHERE id = ?`, "local").Scan(&requested).Error; err != nil {
		t.Fatalf("read flag: %v", err)
	}
	if requested {
		t.Fatal("existing row must default hardware_refresh_requested to false")
	}
}

func TestRenameLegacy_IdempotentOnFreshSchema(t *testing.T) {
	gdb := openRenameTestDB(t)
	if err := gdb.Exec(`CREATE TABLE edges (id TEXT PRIMARY KEY, base_url TEXT)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := gdb.Exec(`CREATE TABLE tasks (id TEXT PRIMARY KEY, edge_id TEXT)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := RenameLegacy(gdb); err != nil {
		t.Fatal(err)
	}
	if !gdb.Migrator().HasTable("edges") {
		t.Fatal("edges missing")
	}
	if !gdb.Migrator().HasColumn("tasks", "edge_id") {
		t.Fatal("edge_id missing")
	}
}

func openRenameTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:rename_legacy_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := Open(Options{DSN: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	return gdb
}
