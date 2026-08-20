package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// EdgeRow is the GORM model for the edges table.
type EdgeRow struct {
	ID                       string     `gorm:"primaryKey;size:128"`
	Name                     string     `gorm:"size:256"`
	Description              string     `gorm:"type:text"`
	Enabled                  bool       `gorm:"not null"`
	CapabilitiesJSON         string     `gorm:"column:capabilities_json;type:text;not null"`
	SubscribeTopicsJSON      string     `gorm:"column:subscribe_topics_json;type:text"`
	AgentTokenEnc            string     `gorm:"column:agent_token_enc;type:text"`
	HardwareJSON             string     `gorm:"column:hardware_json;type:text"`
	HardwareRefreshRequested bool       `gorm:"column:hardware_refresh_requested;not null;default:false"`
	StartedAt                *time.Time `gorm:"column:started_at"`
	ComfyVersion             string     `gorm:"column:comfy_version;size:64"`
	CreatedAt                time.Time  `gorm:"not null"`
	UpdatedAt                time.Time  `gorm:"not null"`
}

func (EdgeRow) TableName() string { return "edges" }

// EdgeRepository is a GORM-backed edge.Repository.
type EdgeRepository struct {
	db *gorm.DB
}

// NewEdgeRepository constructs an EdgeRepository.
func NewEdgeRepository(db *gorm.DB) *EdgeRepository {
	return &EdgeRepository{db: db}
}

func (r *EdgeRepository) Upsert(ctx context.Context, rec *edge.Record) error {
	if rec == nil {
		return fmt.Errorf("edge: nil record")
	}
	if rec.ID == "" {
		return fmt.Errorf("edge: empty id")
	}
	row, err := toRow(rec)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "description", "enabled", "capabilities_json", "updated_at"}),
	}).Create(row).Error
}

func (r *EdgeRepository) Get(ctx context.Context, id sharedkernel.EdgeID) (*edge.Record, error) {
	var row EdgeRow
	err := r.db.WithContext(ctx).First(&row, "id = ?", string(id)).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, edge.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return fromRow(row)
}

func (r *EdgeRepository) List(ctx context.Context) ([]*edge.Record, error) {
	var rows []EdgeRow
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*edge.Record, 0, len(rows))
	for _, row := range rows {
		rec, err := fromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, nil
}

func (r *EdgeRepository) Delete(ctx context.Context, id sharedkernel.EdgeID) error {
	res := r.db.WithContext(ctx).Delete(&EdgeRow{}, "id = ?", string(id))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return edge.ErrNotFound
	}
	return nil
}

func (r *EdgeRepository) UpdateAgentTokenEnc(ctx context.Context, id sharedkernel.EdgeID, enc string) error {
	if id == "" {
		return fmt.Errorf("edge: empty id")
	}
	res := r.db.WithContext(ctx).Model(&EdgeRow{}).Where("id = ?", string(id)).Updates(map[string]any{
		"agent_token_enc": enc,
		"updated_at":      time.Now().UTC(),
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return edge.ErrNotFound
	}
	return nil
}

func (r *EdgeRepository) UpdateHardware(ctx context.Context, id sharedkernel.EdgeID, hw edge.Hardware) error {
	if id == "" {
		return fmt.Errorf("edge: empty id")
	}
	hwJSON := ""
	if !edge.HardwareEmpty(hw) {
		raw, err := json.Marshal(hw)
		if err != nil {
			return fmt.Errorf("edge: hardware_json: %w", err)
		}
		hwJSON = string(raw)
	}
	res := r.db.WithContext(ctx).Model(&EdgeRow{}).Where("id = ?", string(id)).Updates(map[string]any{
		"hardware_json":              hwJSON,
		"hardware_refresh_requested": false,
		"updated_at":                 time.Now().UTC(),
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return edge.ErrNotFound
	}
	return nil
}

func (r *EdgeRepository) SetHardwareRefreshRequested(ctx context.Context, id sharedkernel.EdgeID, requested bool) error {
	if id == "" {
		return fmt.Errorf("edge: empty id")
	}
	res := r.db.WithContext(ctx).Model(&EdgeRow{}).Where("id = ?", string(id)).Updates(map[string]any{
		"hardware_refresh_requested": requested,
		"updated_at":                 time.Now().UTC(),
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return edge.ErrNotFound
	}
	return nil
}

// UpdatePresenceInfo stores agent-reported startup time and Comfy version.
// Empty version or nil startedAt keeps the previously stored value.
func (r *EdgeRepository) UpdatePresenceInfo(ctx context.Context, id sharedkernel.EdgeID, startedAt *time.Time, comfyVersion string) error {
	if id == "" {
		return fmt.Errorf("edge: empty id")
	}
	updates := map[string]any{"updated_at": time.Now().UTC()}
	if startedAt != nil {
		updates["started_at"] = *startedAt
	}
	if comfyVersion != "" {
		updates["comfy_version"] = comfyVersion
	}
	res := r.db.WithContext(ctx).Model(&EdgeRow{}).Where("id = ?", string(id)).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return edge.ErrNotFound
	}
	return nil
}

// UpdateSubscribeTopics stores the edge's topic subscription list.
func (r *EdgeRepository) UpdateSubscribeTopics(ctx context.Context, id sharedkernel.EdgeID, topics []string) error {
	if id == "" {
		return fmt.Errorf("edge: empty id")
	}
	raw, err := json.Marshal(topics)
	if err != nil {
		return fmt.Errorf("edge: subscribe_topics_json: %w", err)
	}
	res := r.db.WithContext(ctx).Model(&EdgeRow{}).Where("id = ?", string(id)).Updates(map[string]any{
		"subscribe_topics_json": string(raw),
		"updated_at":            time.Now().UTC(),
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return edge.ErrNotFound
	}
	return nil
}

func toRow(rec *edge.Record) (*EdgeRow, error) {
	caps := rec.Capabilities
	if caps == nil {
		caps = []string{}
	}
	raw, err := json.Marshal(caps)
	if err != nil {
		return nil, err
	}
	topics, err := json.Marshal(rec.SubscribeTopics)
	if err != nil {
		return nil, err
	}
	hwJSON := ""
	if !edge.HardwareEmpty(rec.Hardware) {
		hw, err := json.Marshal(rec.Hardware)
		if err != nil {
			return nil, fmt.Errorf("edge: hardware_json: %w", err)
		}
		hwJSON = string(hw)
	}
	return &EdgeRow{
		ID:                       string(rec.ID),
		Name:                     rec.Name,
		Description:              rec.Description,
		Enabled:                  rec.Enabled,
		CapabilitiesJSON:         string(raw),
		SubscribeTopicsJSON:      string(topics),
		AgentTokenEnc:            rec.AgentTokenEnc,
		HardwareJSON:             hwJSON,
		HardwareRefreshRequested: rec.HardwareRefreshRequested,
		StartedAt:                rec.StartedAt,
		ComfyVersion:             rec.ComfyVersion,
		CreatedAt:                rec.CreatedAt,
		UpdatedAt:                rec.UpdatedAt,
	}, nil
}

func fromRow(row EdgeRow) (*edge.Record, error) {
	var caps []string
	if row.CapabilitiesJSON != "" {
		if err := json.Unmarshal([]byte(row.CapabilitiesJSON), &caps); err != nil {
			return nil, fmt.Errorf("edge: capabilities_json: %w", err)
		}
	}
	var topics []string
	if row.SubscribeTopicsJSON != "" {
		if err := json.Unmarshal([]byte(row.SubscribeTopicsJSON), &topics); err != nil {
			return nil, fmt.Errorf("edge: subscribe_topics_json: %w", err)
		}
	}
	var hw edge.Hardware
	if row.HardwareJSON != "" {
		if err := json.Unmarshal([]byte(row.HardwareJSON), &hw); err != nil {
			return nil, fmt.Errorf("edge: hardware_json: %w", err)
		}
	}
	return &edge.Record{
		ID:                       sharedkernel.EdgeID(row.ID),
		Name:                     row.Name,
		Description:              row.Description,
		Enabled:                  row.Enabled,
		Capabilities:             caps,
		SubscribeTopics:          topics,
		AgentTokenEnc:            row.AgentTokenEnc,
		Hardware:                 hw,
		HardwareRefreshRequested: row.HardwareRefreshRequested,
		StartedAt:                row.StartedAt,
		ComfyVersion:             row.ComfyVersion,
		CreatedAt:                row.CreatedAt,
		UpdatedAt:                row.UpdatedAt,
	}, nil
}

var _ edge.Repository = (*EdgeRepository)(nil)
