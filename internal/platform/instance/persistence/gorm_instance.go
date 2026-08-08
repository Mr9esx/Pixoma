package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// InstanceRow is the GORM model for comfy_instances.
type InstanceRow struct {
	ID               string    `gorm:"primaryKey;size:128"`
	BaseURL          string    `gorm:"column:base_url;size:512;not null"`
	Enabled          bool      `gorm:"not null"`
	CapabilitiesJSON string    `gorm:"column:capabilities_json;type:text;not null"`
	CreatedAt        time.Time `gorm:"not null"`
	UpdatedAt        time.Time `gorm:"not null"`
}

func (InstanceRow) TableName() string { return "comfy_instances" }

// InstanceRepository is a GORM-backed instance.Repository.
type InstanceRepository struct {
	db *gorm.DB
}

// NewInstanceRepository constructs an InstanceRepository.
func NewInstanceRepository(db *gorm.DB) *InstanceRepository {
	return &InstanceRepository{db: db}
}

func (r *InstanceRepository) Upsert(ctx context.Context, rec *instance.Record) error {
	if rec == nil {
		return fmt.Errorf("instance: nil record")
	}
	if rec.ID == "" {
		return fmt.Errorf("instance: empty id")
	}
	if rec.BaseURL == "" {
		return fmt.Errorf("instance: empty base_url")
	}
	row, err := toRow(rec)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"base_url", "enabled", "capabilities_json", "updated_at"}),
	}).Create(row).Error
}

func (r *InstanceRepository) Get(ctx context.Context, id sharedkernel.InstanceID) (*instance.Record, error) {
	var row InstanceRow
	err := r.db.WithContext(ctx).First(&row, "id = ?", string(id)).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, instance.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return fromRow(row)
}

func (r *InstanceRepository) List(ctx context.Context) ([]*instance.Record, error) {
	var rows []InstanceRow
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*instance.Record, 0, len(rows))
	for _, row := range rows {
		rec, err := fromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, nil
}

func (r *InstanceRepository) Delete(ctx context.Context, id sharedkernel.InstanceID) error {
	res := r.db.WithContext(ctx).Delete(&InstanceRow{}, "id = ?", string(id))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return instance.ErrNotFound
	}
	return nil
}

func toRow(rec *instance.Record) (*InstanceRow, error) {
	caps := rec.Capabilities
	if caps == nil {
		caps = []string{}
	}
	raw, err := json.Marshal(caps)
	if err != nil {
		return nil, err
	}
	return &InstanceRow{
		ID:               string(rec.ID),
		BaseURL:          rec.BaseURL,
		Enabled:          rec.Enabled,
		CapabilitiesJSON: string(raw),
		CreatedAt:        rec.CreatedAt,
		UpdatedAt:        rec.UpdatedAt,
	}, nil
}

func fromRow(row InstanceRow) (*instance.Record, error) {
	var caps []string
	if row.CapabilitiesJSON != "" {
		if err := json.Unmarshal([]byte(row.CapabilitiesJSON), &caps); err != nil {
			return nil, fmt.Errorf("instance: capabilities_json: %w", err)
		}
	}
	return &instance.Record{
		ID:           sharedkernel.InstanceID(row.ID),
		BaseURL:      row.BaseURL,
		Enabled:      row.Enabled,
		Capabilities: caps,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}, nil
}

var _ instance.Repository = (*InstanceRepository)(nil)
