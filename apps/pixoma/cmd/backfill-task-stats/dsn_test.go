package main

import "testing"

func TestResolveBackfillDSN(t *testing.T) {
	t.Run("explicit DATABASE_DSN wins and DB_DRIVER applies", func(t *testing.T) {
		t.Setenv("DATABASE_DSN", "file:/tmp/x.db")
		t.Setenv("DATA_DIR", "/tmp/data")
		t.Setenv("DB_DRIVER", "postgres")
		driver, dsn := resolveBackfillDSN()
		if driver != "postgres" || dsn != "file:/tmp/x.db" {
			t.Fatalf("driver=%q dsn=%q", driver, dsn)
		}
	})
	t.Run("falls back to DATA_DIR/app.db", func(t *testing.T) {
		t.Setenv("DATABASE_DSN", "")
		t.Setenv("DATA_DIR", "/tmp/data")
		t.Setenv("DB_DRIVER", "")
		driver, dsn := resolveBackfillDSN()
		if driver != "sqlite" || dsn != "/tmp/data/app.db" {
			t.Fatalf("driver=%q dsn=%q", driver, dsn)
		}
	})
	t.Run("defaults to data/app.db", func(t *testing.T) {
		t.Setenv("DATABASE_DSN", "")
		t.Setenv("DATA_DIR", "")
		t.Setenv("DB_DRIVER", "")
		driver, dsn := resolveBackfillDSN()
		if driver != "sqlite" || dsn != "data/app.db" {
			t.Fatalf("driver=%q dsn=%q", driver, dsn)
		}
	})
}
