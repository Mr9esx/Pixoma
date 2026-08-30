package db_test

import (
	"os"
	"path/filepath"
	"strings"
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

func TestOpen_UnknownDriver(t *testing.T) {
	_, err := db.Open(db.Options{Driver: "oracle", DSN: "x"})
	if err == nil {
		t.Fatal("expected unknown driver error")
	}
	if !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("got %v", err)
	}
}

func TestOpen_MySQLEmptyDSN(t *testing.T) {
	_, err := db.Open(db.Options{Driver: db.DriverMySQL, DSN: ""})
	if err == nil {
		t.Fatal("expected empty DSN error")
	}
}

func TestOpen_MySQLUnreachable(t *testing.T) {
	_, err := db.Open(db.Options{
		Driver: db.DriverMySQL,
		DSN:    "pixoma:pixoma@tcp(127.0.0.1:1)/pixoma?timeout=1s&readTimeout=1s&writeTimeout=1s",
	})
	if err == nil {
		t.Fatal("expected mysql dial to fail")
	}
}

func TestOpen_PostgresUnreachable(t *testing.T) {
	_, err := db.Open(db.Options{
		Driver: db.DriverPostgres,
		DSN:    "host=127.0.0.1 port=1 user=pixoma password=pixoma dbname=pixoma sslmode=disable connect_timeout=1",
	})
	if err == nil {
		t.Fatal("expected postgres dial to fail")
	}
}

func TestOpenSQLiteUsesPrivateFilePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.db")
	gdb, err := db.Open(db.Options{DSN: "file:" + path + "?cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 0600", info.Mode().Perm())
	}
}
