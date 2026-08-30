package db

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	DriverSQLite   = "sqlite"
	DriverMySQL    = "mysql"
	DriverPostgres = "postgres"
)

type Options struct {
	Driver string // sqlite (default), mysql, postgres
	DSN    string
	Debug  bool
}

func Open(opts Options) (*gorm.DB, error) {
	if opts.DSN == "" {
		return nil, fmt.Errorf("db: empty DSN")
	}
	cfg := &gorm.Config{
		Logger: newGormLogger(opts.Debug, os.Stdout),
	}
	dial, err := dialector(opts.Driver, opts.DSN)
	if err != nil {
		return nil, err
	}
	gdb, err := gorm.Open(dial, cfg)
	if err != nil {
		return nil, fmt.Errorf("db: open: %w", err)
	}
	if isSQLite(opts.Driver) {
		if path, ok := sqliteFilePath(opts.DSN); ok {
			if err := os.Chmod(path, 0o600); err != nil {
				_ = closeDB(gdb)
				return nil, fmt.Errorf("db: secure sqlite file: %w", err)
			}
		}
	}
	if sqlDB, err := gdb.DB(); err == nil {
		sqlDB.SetConnMaxLifetime(time.Hour)
		sqlDB.SetMaxIdleConns(4)
		sqlDB.SetMaxOpenConns(16)
	}
	return gdb, nil
}

func isSQLite(driver string) bool {
	return strings.EqualFold(strings.TrimSpace(driver), "") || strings.EqualFold(strings.TrimSpace(driver), DriverSQLite)
}

func sqliteFilePath(dsn string) (string, bool) {
	raw := strings.TrimSpace(dsn)
	if raw == "" || strings.Contains(raw, ":memory:") {
		return "", false
	}
	query := ""
	path := raw
	if before, after, found := strings.Cut(raw, "?"); found {
		path, query = before, after
	}
	path = strings.TrimPrefix(path, "file:")
	if strings.Contains(query, "mode=memory") {
		return "", false
	}
	if path == "" {
		return "", false
	}
	return path, true
}

func newGormLogger(debug bool, w io.Writer) logger.Interface {
	level := logger.Warn
	if debug {
		level = logger.Info
	}
	return logger.New(log.New(w, "\r\n", log.LstdFlags), logger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  level,
		IgnoreRecordNotFoundError: true,
		Colorful:                  false,
	})
}

func closeDB(gdb *gorm.DB) error {
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func dialector(driver, dsn string) (gorm.Dialector, error) {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "", DriverSQLite:
		return sqlite.Open(dsn), nil
	case DriverMySQL:
		return mysql.Open(dsn), nil
	case DriverPostgres:
		return postgres.Open(dsn), nil
	default:
		return nil, fmt.Errorf("db: unknown driver %q", driver)
	}
}
