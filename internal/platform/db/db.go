package db

import (
	"fmt"
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
	cfg := &gorm.Config{}
	if !opts.Debug {
		cfg.Logger = logger.Default.LogMode(logger.Warn)
	}
	dial, err := dialector(opts.Driver, opts.DSN)
	if err != nil {
		return nil, err
	}
	gdb, err := gorm.Open(dial, cfg)
	if err != nil {
		return nil, fmt.Errorf("db: open: %w", err)
	}
	if sqlDB, err := gdb.DB(); err == nil {
		sqlDB.SetConnMaxLifetime(time.Hour)
		sqlDB.SetMaxIdleConns(4)
		sqlDB.SetMaxOpenConns(16)
	}
	return gdb, nil
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
