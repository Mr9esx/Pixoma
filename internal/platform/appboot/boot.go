package appboot

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance"
	instpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/instance/persistence"
)

// Options configures shared DB open, migrate, and optional instance seeding.
type Options struct {
	Driver string // sqlite (default), mysql, postgres
	DSN    string
	Debug  bool

	// MigrateInstances AutoMigrates InstanceRow when true.
	MigrateInstances bool

	// Models are additional GORM models to AutoMigrate (e.g. bot Case/User rows).
	Models []any

	// Seed, when non-nil, upserts comfy_instances via instance.SeedFromConfig.
	Seed *instance.SeedConfig
}

// Bootstrap opens the database, runs AutoMigrate for requested models, and
// optionally seeds comfy instances. cleanup closes the underlying sql.DB.
func Bootstrap(ctx context.Context, opts Options) (*gorm.DB, func() error, error) {
	gdb, err := Open(opts)
	if err != nil {
		return nil, nil, err
	}

	models := append([]any(nil), opts.Models...)
	if opts.MigrateInstances {
		models = append(models, &instpersist.InstanceRow{})
	}
	if len(models) > 0 {
		if err := Migrate(gdb, models...); err != nil {
			_ = closeDB(gdb)
			return nil, nil, err
		}
	}

	if opts.Seed != nil {
		repo := instpersist.NewInstanceRepository(gdb)
		if _, err := instance.SeedFromConfig(ctx, repo, *opts.Seed); err != nil {
			_ = closeDB(gdb)
			return nil, nil, fmt.Errorf("appboot: seed instances: %w", err)
		}
	}

	return gdb, func() error { return closeDB(gdb) }, nil
}

// Open opens a GORM DB using the shared platform/db helper.
func Open(opts Options) (*gorm.DB, error) {
	gdb, err := db.Open(db.Options{Driver: opts.Driver, DSN: opts.DSN, Debug: opts.Debug})
	if err != nil {
		return nil, fmt.Errorf("appboot: %w", err)
	}
	return gdb, nil
}

// Migrate runs GORM AutoMigrate for the given models.
func Migrate(gdb *gorm.DB, models ...any) error {
	if err := db.AutoMigrate(gdb, models...); err != nil {
		return fmt.Errorf("appboot: %w", err)
	}
	return nil
}

func closeDB(gdb *gorm.DB) error {
	if gdb == nil {
		return nil
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
