package db_test

import (
	"strings"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
)

func TestEnsureDatabase_SQLiteNoop(t *testing.T) {
	if err := db.EnsureDatabase(db.DriverSQLite, "data/app.db"); err != nil {
		t.Fatalf("sqlite ensure must be a no-op, got %v", err)
	}
}

func TestEnsureDatabase_UnknownDriver(t *testing.T) {
	err := db.EnsureDatabase("oracle", "x")
	if err == nil || !strings.Contains(err.Error(), "unknown driver") {
		t.Fatalf("want unknown driver error, got %v", err)
	}
}

func TestEnsureDatabase_MySQLMissingDBName(t *testing.T) {
	err := db.EnsureDatabase(db.DriverMySQL, "user:pass@tcp(127.0.0.1:3306)/")
	if err == nil || !strings.Contains(err.Error(), "missing database name") {
		t.Fatalf("want missing database name error, got %v", err)
	}
}

func TestEnsureDatabase_PostgresMissingDBName(t *testing.T) {
	err := db.EnsureDatabase(db.DriverPostgres, "host=127.0.0.1 user=pixoma password=secret")
	if err == nil || !strings.Contains(err.Error(), "missing database name") {
		t.Fatalf("want missing database name error, got %v", err)
	}
}

func TestEnsureDatabase_PostgresInvalidDSN(t *testing.T) {
	err := db.EnsureDatabase(db.DriverPostgres, "not a dsn at all")
	if err == nil {
		t.Fatal("want parse error for invalid postgres dsn")
	}
}
