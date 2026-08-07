package db

import (
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Options struct {
	DSN   string
	Debug bool
}

func Open(opts Options) (*gorm.DB, error) {
	if opts.DSN == "" {
		return nil, fmt.Errorf("db: empty DSN")
	}
	cfg := &gorm.Config{}
	if !opts.Debug {
		cfg.Logger = logger.Default.LogMode(logger.Warn)
	}
	gdb, err := gorm.Open(sqlite.Open(opts.DSN), cfg)
	if err != nil {
		return nil, fmt.Errorf("db: open: %w", err)
	}
	return gdb, nil
}
