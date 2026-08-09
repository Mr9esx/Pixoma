package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type CaseRow struct {
	ID        string `gorm:"primaryKey;size:128"`
	Name      string `gorm:"size:256;not null"`
	MenuKey   string `gorm:"size:128;index"`
	TagsJSON  string `gorm:"type:text"`
	CatsJSON  string `gorm:"type:text"`
	DocJSON   string `gorm:"type:text;not null"`
	Enabled   bool   `gorm:"not null;default:true;index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (CaseRow) TableName() string { return "catalog_cases" }

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) Create(ctx context.Context, c *domain.Case) error {
	if c == nil {
		return fmt.Errorf("nil case")
	}
	var n int64
	if err := r.db.WithContext(ctx).Model(&CaseRow{}).Where("id = ?", string(c.Document.ID)).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return domain.ErrAlreadyExists
	}
	row, err := toRow(c)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *GormRepository) Save(ctx context.Context, c *domain.Case) error {
	if c == nil {
		return fmt.Errorf("nil case")
	}
	row, err := toRow(c)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *GormRepository) Get(ctx context.Context, id sharedkernel.CaseID) (*domain.Case, error) {
	var row CaseRow
	err := r.db.WithContext(ctx).First(&row, "id = ?", string(id)).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return fromRow(row)
}

func (r *GormRepository) List(ctx context.Context, q domain.ListQuery) ([]*domain.Case, error) {
	tx := r.db.WithContext(ctx).Model(&CaseRow{})
	if q.Enabled != nil {
		tx = tx.Where("enabled = ?", *q.Enabled)
	}
	if q.MenuKey != "" {
		tx = tx.Where("menu_key = ?", q.MenuKey)
	}
	if q.Tag != "" {
		tx = tx.Where("tags_json LIKE ?", "%\""+q.Tag+"\"%")
	}
	if q.Category != "" {
		tx = tx.Where("cats_json LIKE ?", "%\""+q.Category+"\"%")
	}
	if q.Q != "" {
		like := "%" + q.Q + "%"
		tx = tx.Where("id LIKE ? OR name LIKE ? OR menu_key LIKE ?", like, like, like)
	}
	if q.CreatedFrom != nil {
		tx = tx.Where("created_at >= ?", *q.CreatedFrom)
	}
	if q.CreatedTo != nil {
		tx = tx.Where("created_at <= ?", *q.CreatedTo)
	}
	if q.Limit > 0 {
		tx = tx.Limit(q.Limit)
	}
	if q.Offset > 0 {
		tx = tx.Offset(q.Offset)
	}
	var rows []CaseRow
	if err := tx.Order("id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.Case, 0, len(rows))
	for _, row := range rows {
		c, err := fromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func (r *GormRepository) Disable(ctx context.Context, id sharedkernel.CaseID) error {
	res := r.db.WithContext(ctx).Model(&CaseRow{}).Where("id = ?", string(id)).Update("enabled", false)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *GormRepository) Enable(ctx context.Context, id sharedkernel.CaseID) error {
	res := r.db.WithContext(ctx).Model(&CaseRow{}).Where("id = ?", string(id)).Update("enabled", true)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func toRow(c *domain.Case) (*CaseRow, error) {
	docBytes, err := json.Marshal(c.Document)
	if err != nil {
		return nil, err
	}
	tags, err := json.Marshal(c.Document.Tags)
	if err != nil {
		return nil, err
	}
	cats, err := json.Marshal(c.Document.Categories)
	if err != nil {
		return nil, err
	}
	return &CaseRow{
		ID:       string(c.Document.ID),
		Name:     c.Document.Name,
		MenuKey:  c.Document.MenuKey,
		TagsJSON: string(tags),
		CatsJSON: string(cats),
		DocJSON:  string(docBytes),
		Enabled:  c.Enabled,
	}, nil
}

func fromRow(row CaseRow) (*domain.Case, error) {
	var doc domain.CaseDocument
	if err := json.Unmarshal([]byte(row.DocJSON), &doc); err != nil {
		return nil, fmt.Errorf("decode case doc: %w", err)
	}
	return &domain.Case{Document: doc, Enabled: row.Enabled}, nil
}
