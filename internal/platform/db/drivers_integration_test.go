//go:build integration

package db_test

import (
	"os"
	"strings"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
)

func TestOpen_MySQLFromEnv(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("PIXOMA_MYSQL_DSN"))
	if dsn == "" {
		t.Skip("set PIXOMA_MYSQL_DSN to run")
	}
	gdb, err := db.Open(db.Options{Driver: db.DriverMySQL, DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := sqlDB.Ping(); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &pingRow{}); err != nil {
		t.Fatal(err)
	}
}

func TestOpen_PostgresFromEnv(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("PIXOMA_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("set PIXOMA_POSTGRES_DSN to run")
	}
	gdb, err := db.Open(db.Options{Driver: db.DriverPostgres, DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := sqlDB.Ping(); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &pingRow{}); err != nil {
		t.Fatal(err)
	}
}

type pingRow struct {
	ID string `gorm:"primaryKey;size:8"`
}

func (pingRow) TableName() string { return "pixoma_integration_ping" }
