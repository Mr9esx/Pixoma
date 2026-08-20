package persistence

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	mcdomain "github.com/mr9esx/comfyui_tgbot/internal/menucard/domain"
)

// MainMenuRow stores the main keyboard menu as a JSON document per channel.
type MainMenuRow struct {
	ChannelID string `gorm:"primaryKey;size:64"`
	DocJSON   string `gorm:"type:text;not null"`
}

func (MainMenuRow) TableName() string { return "channel_main_menus" }

// CardRow stores a card as a JSON document, scoped to a channel.
type CardRow struct {
	ID        string `gorm:"primaryKey;size:64"`
	ChannelID string `gorm:"index;size:64;not null"`
	DocJSON   string `gorm:"type:text;not null"`
}

func (CardRow) TableName() string { return "channel_cards" }

// CardRepository is the persistence port for the new menu/card model.
type CardRepository interface {
	GetMenu(ctx context.Context, channelID string) (mcdomain.Menu, error)
	PutMenu(ctx context.Context, channelID string, menu mcdomain.Menu) error
	ListCards(ctx context.Context, channelID string) ([]mcdomain.Card, error)
	GetCard(ctx context.Context, channelID, id string) (mcdomain.Card, error)
	CreateCard(ctx context.Context, channelID string, card mcdomain.Card) error
	UpdateCard(ctx context.Context, channelID string, card mcdomain.Card) error
	DeleteCard(ctx context.Context, channelID, id string) error
	CardReferences(ctx context.Context, channelID, id string) ([]string, error)
	WorkflowPlacements(ctx context.Context, workflowID string) ([]WorkflowPlacement, error)
}

// WorkflowPlacement is where an open_workflow action references a workflow.
type WorkflowPlacement struct {
	ChannelID string
	ItemID    string
	Label     string
	Kind      string // menu_item | card_button
}

// GormCardRepository implements CardRepository with GORM.
type GormCardRepository struct {
	db *gorm.DB
}

func NewGormCardRepository(db *gorm.DB) *GormCardRepository {
	return &GormCardRepository{db: db}
}

func (r *GormCardRepository) GetMenu(ctx context.Context, channelID string) (mcdomain.Menu, error) {
	var row MainMenuRow
	if err := r.db.WithContext(ctx).First(&row, "channel_id = ?", channelID).Error; err != nil {
		return mcdomain.Menu{}, err
	}
	var menu mcdomain.Menu
	if err := json.Unmarshal([]byte(row.DocJSON), &menu); err != nil {
		return mcdomain.Menu{}, fmt.Errorf("decode menu: %w", err)
	}
	return menu, nil
}

func (r *GormCardRepository) PutMenu(ctx context.Context, channelID string, menu mcdomain.Menu) error {
	raw, err := json.Marshal(menu)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(&MainMenuRow{ChannelID: channelID, DocJSON: string(raw)}).Error
}

func (r *GormCardRepository) ListCards(ctx context.Context, channelID string) ([]mcdomain.Card, error) {
	var rows []CardRow
	if err := r.db.WithContext(ctx).Where("channel_id = ?", channelID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]mcdomain.Card, 0, len(rows))
	for _, row := range rows {
		var card mcdomain.Card
		if err := json.Unmarshal([]byte(row.DocJSON), &card); err != nil {
			return nil, fmt.Errorf("decode card: %w", err)
		}
		out = append(out, card)
	}
	return out, nil
}

func (r *GormCardRepository) GetCard(ctx context.Context, channelID, id string) (mcdomain.Card, error) {
	var row CardRow
	if err := r.db.WithContext(ctx).First(&row, "id = ? AND channel_id = ?", id, channelID).Error; err != nil {
		return mcdomain.Card{}, err
	}
	var card mcdomain.Card
	if err := json.Unmarshal([]byte(row.DocJSON), &card); err != nil {
		return mcdomain.Card{}, fmt.Errorf("decode card: %w", err)
	}
	return card, nil
}

func (r *GormCardRepository) CreateCard(ctx context.Context, channelID string, card mcdomain.Card) error {
	raw, err := json.Marshal(card)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(&CardRow{ID: card.ID, ChannelID: channelID, DocJSON: string(raw)}).Error
}

func (r *GormCardRepository) UpdateCard(ctx context.Context, channelID string, card mcdomain.Card) error {
	raw, err := json.Marshal(card)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).
		Model(&CardRow{}).
		Where("id = ? AND channel_id = ?", card.ID, channelID).
		Update("doc_json", string(raw)).Error
}

func (r *GormCardRepository) DeleteCard(ctx context.Context, channelID, id string) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND channel_id = ?", id, channelID).
		Delete(&CardRow{}).Error
}

func (r *GormCardRepository) CardReferences(ctx context.Context, channelID, id string) ([]string, error) {
	menu, err := r.GetMenu(ctx, channelID)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	refs := []string{}
	for _, it := range menu.Items {
		if it.Action.Type == "open_card" && it.Action.CardID == id {
			refs = append(refs, "menu:"+it.ID)
		}
	}
	cards, err := r.ListCards(ctx, channelID)
	if err != nil {
		return nil, err
	}
	for _, card := range cards {
		for _, b := range card.Buttons {
			if b.Action.Type == "open_card" && b.Action.CardID == id {
				refs = append(refs, "card:"+card.ID+":"+b.ID)
			}
		}
	}
	return refs, nil
}

func (r *GormCardRepository) WorkflowPlacements(ctx context.Context, workflowID string) ([]WorkflowPlacement, error) {
	var out []WorkflowPlacement
	var menus []MainMenuRow
	if err := r.db.WithContext(ctx).Find(&menus).Error; err != nil {
		return nil, err
	}
	for _, row := range menus {
		var menu mcdomain.Menu
		if err := json.Unmarshal([]byte(row.DocJSON), &menu); err != nil {
			return nil, fmt.Errorf("decode menu: %w", err)
		}
		for _, it := range menu.Items {
			if actionContainsWorkflow(it.Action, workflowID) {
				out = append(out, WorkflowPlacement{ChannelID: row.ChannelID, ItemID: it.ID, Label: it.Label, Kind: "menu_item"})
			}
		}
	}
	var cards []CardRow
	if err := r.db.WithContext(ctx).Find(&cards).Error; err != nil {
		return nil, err
	}
	for _, row := range cards {
		var card mcdomain.Card
		if err := json.Unmarshal([]byte(row.DocJSON), &card); err != nil {
			return nil, fmt.Errorf("decode card: %w", err)
		}
		for _, b := range card.Buttons {
			if actionContainsWorkflow(b.Action, workflowID) {
				out = append(out, WorkflowPlacement{ChannelID: row.ChannelID, ItemID: b.ID, Label: b.Label, Kind: "card_button"})
			}
		}
	}
	return out, nil
}

func actionContainsWorkflow(a mcdomain.Action, workflowID string) bool {
	if a.Type != "open_workflow" {
		return false
	}
	for _, id := range a.WorkflowIDs {
		if id == workflowID {
			return true
		}
	}
	return false
}
