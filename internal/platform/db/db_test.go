package db_test

import (
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
)

func TestOpenMemorySQLite(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file::memory:?cache=shared"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatalf("sql db: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}
}
