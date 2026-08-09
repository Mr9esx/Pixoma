package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/mr9esx/comfyui_tgbot/internal/tgmenu/domain"
)

type MenuRow struct {
	ID        string    `gorm:"primaryKey;size:64"`
	ItemsJSON string    `gorm:"type:text;not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

func (MenuRow) TableName() string { return "tg_menu_configs" }

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) Get(ctx context.Context, id string) (domain.MenuDocument, error) {
	var row MenuRow
	err := r.db.WithContext(ctx).First(&row, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.MenuDocument{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.MenuDocument{}, err
	}
	var items []domain.MenuItem
	if err := json.Unmarshal([]byte(row.ItemsJSON), &items); err != nil {
		return domain.MenuDocument{}, fmt.Errorf("unmarshal menu items: %w", err)
	}
	return domain.MenuDocument{
		ID:        row.ID,
		Items:     items,
		UpdatedAt: row.UpdatedAt,
	}, nil
}

func (r *GormRepository) Replace(ctx context.Context, doc domain.MenuDocument) error {
	if doc.ID == "" {
		doc.ID = domain.DocumentIDDefault
	}
	b, err := json.Marshal(doc.Items)
	if err != nil {
		return fmt.Errorf("marshal menu items: %w", err)
	}
	row := MenuRow{
		ID:        doc.ID,
		ItemsJSON: string(b),
		UpdatedAt: time.Now().UTC(),
	}
	return r.db.WithContext(ctx).Save(&row).Error
}

func (r *GormRepository) EnsureDefault(ctx context.Context) (domain.MenuDocument, error) {
	doc, err := r.Get(ctx, domain.DocumentIDDefault)
	if err == nil {
		return doc, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return domain.MenuDocument{}, err
	}
	seed := domain.DefaultSeed()
	if err := r.Replace(ctx, seed); err != nil {
		return domain.MenuDocument{}, err
	}
	return r.Get(ctx, domain.DocumentIDDefault)
}
