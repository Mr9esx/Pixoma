package persistence

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	mcdomain "github.com/Mr9esx/Pixoma/internal/menus/domain"
)

// MainMenuRow stores the menu tree as a JSON document per channel.
type MainMenuRow struct {
	ChannelID string `gorm:"primaryKey;size:64"`
	DocJSON   string `gorm:"type:text;not null"`
}

func (MainMenuRow) TableName() string { return "channel_main_menus" }

// CardRow is leftover from the old card table; channel delete may still drop rows.
type CardRow struct {
	ID        string `gorm:"primaryKey;size:64"`
	ChannelID string `gorm:"index;size:64;not null"`
	DocJSON   string `gorm:"type:text;not null"`
}

func (CardRow) TableName() string { return "channel_cards" }

// CardRepository persists a nested menu tree per channel.
type CardRepository interface {
	GetTree(ctx context.Context, channelID string) (mcdomain.MenuTree, error)
	PutTree(ctx context.Context, channelID string, tree mcdomain.MenuTree) error
	WorkflowPlacements(ctx context.Context, workflowID string) ([]WorkflowPlacement, error)
	RemoveWorkflowReferences(ctx context.Context, workflowID string) ([]WorkflowPlacement, error)
}

// WorkflowPlacement is where an open_workflow action appears in a tree.
type WorkflowPlacement struct {
	ChannelID   string
	ChannelName string
	ItemID      string
	Label       string
	Kind        string // keyboard | card_button
	Labels      []string
}

// GormCardRepository implements CardRepository with GORM.
type GormCardRepository struct {
	db *gorm.DB
}

func NewGormCardRepository(db *gorm.DB) *GormCardRepository {
	return &GormCardRepository{db: db}
}

func (r *GormCardRepository) GetTree(ctx context.Context, channelID string) (mcdomain.MenuTree, error) {
	var row MainMenuRow
	if err := r.db.WithContext(ctx).First(&row, "channel_id = ?", channelID).Error; err != nil {
		return mcdomain.MenuTree{}, err
	}
	tree, ok := mcdomain.ParseStoredMenuTree([]byte(row.DocJSON), channelID)
	if !ok {
		return mcdomain.MenuTree{}, gorm.ErrRecordNotFound
	}
	return tree, nil
}

func (r *GormCardRepository) PutTree(ctx context.Context, channelID string, tree mcdomain.MenuTree) error {
	tree.ID = channelID
	raw, err := json.Marshal(tree)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(&MainMenuRow{ChannelID: channelID, DocJSON: string(raw)}).Error
}

func (r *GormCardRepository) WorkflowPlacements(ctx context.Context, workflowID string) ([]WorkflowPlacement, error) {
	var menus []MainMenuRow
	if err := r.db.WithContext(ctx).Find(&menus).Error; err != nil {
		return nil, err
	}
	var out []WorkflowPlacement
	for _, row := range menus {
		tree, ok := mcdomain.ParseStoredMenuTree([]byte(row.DocJSON), row.ChannelID)
		if !ok {
			continue
		}
		for _, p := range mcdomain.WalkWorkflowPlacements(tree) {
			if p.WorkflowID != workflowID {
				continue
			}
			out = append(out, WorkflowPlacement{
				ChannelID: row.ChannelID,
				ItemID:    p.ButtonID,
				Label:     p.Labels[len(p.Labels)-1],
				Kind:      p.Kind,
				Labels:    p.Labels,
			})
		}
	}
	if len(out) > 0 {
		if err := r.fillChannelNames(ctx, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (r *GormCardRepository) fillChannelNames(ctx context.Context, placements []WorkflowPlacement) error {
	seen := make(map[string]struct{}, len(placements))
	ids := make([]string, 0, len(placements))
	for _, p := range placements {
		if _, ok := seen[p.ChannelID]; ok {
			continue
		}
		seen[p.ChannelID] = struct{}{}
		ids = append(ids, p.ChannelID)
	}
	type channelNameRow struct {
		ID   string
		Name string
	}
	var rows []channelNameRow
	if err := r.db.WithContext(ctx).
		Table("channels").
		Select("id, name").
		Where("id IN ?", ids).
		Scan(&rows).Error; err != nil {
		return fmt.Errorf("load channel names: %w", err)
	}
	byID := make(map[string]string, len(rows))
	for _, row := range rows {
		byID[row.ID] = row.Name
	}
	for i := range placements {
		placements[i].ChannelName = byID[placements[i].ChannelID]
	}
	return nil
}

func (r *GormCardRepository) RemoveWorkflowReferences(ctx context.Context, workflowID string) ([]WorkflowPlacement, error) {
	var menus []MainMenuRow
	if err := r.db.WithContext(ctx).Find(&menus).Error; err != nil {
		return nil, err
	}
	var removed []WorkflowPlacement
	for _, row := range menus {
		tree, ok := mcdomain.ParseStoredMenuTree([]byte(row.DocJSON), row.ChannelID)
		if !ok {
			continue
		}
		got, hits := stripWorkflowButtons(tree, workflowID)
		if len(hits) == 0 {
			continue
		}
		for i := range hits {
			hits[i].ChannelID = row.ChannelID
		}
		removed = append(removed, hits...)
		if len(got.Items) == 0 {
			got = mcdomain.DefaultMenuTree(row.ChannelID)
		}
		if err := r.PutTree(ctx, row.ChannelID, got); err != nil {
			return nil, err
		}
	}
	if len(removed) > 0 {
		if err := r.fillChannelNames(ctx, removed); err != nil {
			return nil, err
		}
	}
	return removed, nil
}

func stripWorkflowButtons(tree mcdomain.MenuTree, workflowID string) (mcdomain.MenuTree, []WorkflowPlacement) {
	var removed []WorkflowPlacement
	tree.Items = filterButtons(tree.Items, workflowID, nil, "keyboard", &removed)
	return tree, removed
}

func filterButtons(items []mcdomain.TreeButton, workflowID string, prefix []string, kind string, removed *[]WorkflowPlacement) []mcdomain.TreeButton {
	out := make([]mcdomain.TreeButton, 0, len(items))
	for _, b := range items {
		path := append(append([]string{}, prefix...), b.Label)
		if b.Action.Type == "open_workflow" && b.Action.WorkflowID == workflowID {
			*removed = append(*removed, WorkflowPlacement{
				ItemID: b.ID, Label: b.Label, Kind: kind, Labels: path,
			})
			continue
		}
		if b.Action.Type == "open_card" && b.Action.Card != nil {
			card := *b.Action.Card
			card.Buttons = filterButtons(card.Buttons, workflowID, path, "card_button", removed)
			b.Action.Card = &card
		}
		out = append(out, b)
	}
	return out
}