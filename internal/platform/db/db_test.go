package db_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/platform/db"
	"gorm.io/gorm"
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

func TestOpenSQLiteSerializesReadThenWriteTransactions(t *testing.T) {
	if err := os.MkdirAll(".local-cache", 0o700); err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp(".local-cache", "sqlite-transactions-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	gdb, err := db.Open(db.Options{DSN: filepath.Join(dir, "app.db")})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := gdb.Exec("CREATE TABLE context_events (id INTEGER PRIMARY KEY)").Error; err != nil {
		t.Fatal(err)
	}

	first := gdb.Begin()
	if first.Error != nil {
		t.Fatal(first.Error)
	}
	defer first.Rollback()
	var count int64
	if err := first.Raw("SELECT COUNT(*) FROM context_events").Scan(&count).Error; err != nil {
		t.Fatal(err)
	}

	readDone := make(chan struct{})
	continueWrite := make(chan struct{})
	secondDone := make(chan error, 1)
	go func() {
		secondDone <- gdb.Transaction(func(second *gorm.DB) error {
			var secondCount int64
			if err := second.Raw("SELECT COUNT(*) FROM context_events").Scan(&secondCount).Error; err != nil {
				return err
			}
			close(readDone)
			<-continueWrite
			return second.Exec("INSERT INTO context_events (id) VALUES (2)").Error
		})
	}()
	select {
	case <-readDone:
	case <-time.After(100 * time.Millisecond):
	}
	if err := first.Exec("INSERT INTO context_events (id) VALUES (1)").Error; err != nil {
		t.Fatal(err)
	}
	close(continueWrite)
	if err := first.Commit().Error; err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-secondDone:
		if err != nil {
			t.Fatalf("second transaction: %v", err)
		}
	case <-time.After(7 * time.Second):
		t.Fatal("second transaction did not finish")
	}
	if err := gdb.Raw("SELECT COUNT(*) FROM context_events").Scan(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("events = %d, want 2", count)
	}
}
