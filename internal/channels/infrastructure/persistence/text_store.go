package persistence

import (
	"context"
	"fmt"

	"github.com/Mr9esx/Pixoma/internal/channels/domain/templates"
	"gorm.io/gorm"
)

// Row is a stored copy-template override. Rows exist for the platform default
// (ChannelID == "") and per channel (ChannelID == a channel id). Templates that
// have no row fall back to the built-in default.
type Row struct {
	ChannelID string `gorm:"column:channel_id;size:64;primaryKey;not null;default:''"`
	Key       string `gorm:"primaryKey;size:64"`
	Template  string `gorm:"column:template;type:text;not null"`
}

// TableName returns the database table backing copy-template overrides.
func (Row) TableName() string { return "text_templates" }

// GlobalDefaultID identifies platform-default copy rows (seed source for new
// channels and the render fallback after a channel's own override).
const GlobalDefaultID = ""

// Store persists copy-template overrides in the business database.
type Store struct {
	db *gorm.DB
}

// NewStore returns a Store backed by the given database, migrating the table.
func NewStore(gdb *gorm.DB) (*Store, error) {
	if gdb == nil {
		return nil, fmt.Errorf("text: nil db")
	}
	if err := gdb.AutoMigrate(&Row{}); err != nil {
		return nil, fmt.Errorf("text: migrate: %w", err)
	}
	return &Store{db: gdb}, nil
}

// Render returns the effective template for a channel, falling back to the
// platform default and then the built-in default, then interpolates vars.
func (s *Store) Render(ctx context.Context, channelID, key string, vars map[string]string) string {
	if s == nil {
		return templates.Render(templates.Default(key), vars)
	}
	var row Row
	// channel override
	err := s.db.WithContext(ctx).
		Where("`channel_id` = ? AND `key` = ? AND `template` <> ''", channelID, key).
		Take(&row).Error
	if err == nil {
		return templates.Render(row.Template, vars)
	}
	// platform default
	err = s.db.WithContext(ctx).
		Where("`channel_id` = ? AND `key` = ? AND `template` <> ''", GlobalDefaultID, key).
		Take(&row).Error
	if err == nil {
		return templates.Render(row.Template, vars)
	}
	return templates.Render(templates.Default(key), vars)
}

// Load returns every stored override for a scope, keyed by template key.
func (s *Store) Load(ctx context.Context, channelID string) (map[string]string, error) {
	if s == nil {
		return map[string]string{}, nil
	}
	var rows []Row
	if err := s.db.WithContext(ctx).
		Where("`channel_id` = ?", channelID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, r := range rows {
		out[r.Key] = r.Template
	}
	return out, nil
}

// Save upserts the given overrides for a scope. Empty values are treated as
// "reset to fallback" and delete the stored row.
func (s *Store) Save(ctx context.Context, channelID string, overrides map[string]string) error {
	if s == nil {
		return nil
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for k, v := range overrides {
			if !templates.HasKey(k) {
				continue // unknown keys are ignored
			}
			if v == "" {
				if err := tx.Where("`channel_id` = ? AND `key` = ?", channelID, k).
					Delete(&Row{}).Error; err != nil {
					return err
				}
				continue
			}
			row := Row{ChannelID: channelID, Key: k, Template: v}
			if err := tx.Save(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// Reset clears the overrides for a subset of keys (empty slice clears none).
func (s *Store) Reset(ctx context.Context, channelID string, keys []string) error {
	if s == nil || len(keys) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).
		Where("`channel_id` = ? AND `key` IN ?", channelID, keys).
		Delete(&Row{}).Error
}

// Seed materializes the effective defaults (platform defaults with built-in
// fallback) as an explicit per-channel copy. Used when a channel is created so
// every channel owns an independent set of templates.
func (s *Store) Seed(ctx context.Context, channelID string) error {
	if s == nil {
		return nil
	}
	overrides, err := s.Load(ctx, GlobalDefaultID)
	if err != nil {
		return err
	}
	effective := make(map[string]string, len(templates.Keys()))
	for _, k := range templates.Keys() {
		effective[k] = chained(overrides[k], k)
	}
	return s.Save(ctx, channelID, effective)
}

// chained resolves a key to its effective default: stored override first, else
// the built-in default.
func chained(stored, key string) string {
	if stored != "" {
		return stored
	}
	return templates.Default(key)
}
