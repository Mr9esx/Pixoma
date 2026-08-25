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
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	Name      string `gorm:"size:256;not null"`
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
	if c.Document.ID == 0 {
		// id=0 时显式分配下一个 id，不依赖数据库自增回填：
		// 旧 SQLite 表（TEXT 主键）会把 0 写成 NULL，导致 create 后 Get 找不到行。
		next, err := r.nextCaseID(ctx)
		if err != nil {
			return err
		}
		c.Document.ID = sharedkernel.CaseID(next)
	}
	var n int64
	if err := r.db.WithContext(ctx).Model(&CaseRow{}).Where("id = ?", c.Document.ID).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return domain.ErrAlreadyExists
	}
	row, err := toRow(c)
	if err != nil {
		return err
	}
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return err
	}
	// id=0 时由数据库自动分配：回写调用方并同步文档内嵌 id，
	// 保证回读（fromRow）时 CaseDocument.ID 与行主键一致。
	if c.Document.ID == 0 && row.ID != 0 {
		c.Document.ID = sharedkernel.CaseID(row.ID)
		docBytes, err := json.Marshal(c.Document)
		if err != nil {
			return err
		}
		if err := r.db.WithContext(ctx).
			Model(&CaseRow{}).
			Where("id = ?", row.ID).
			Update("doc_json", string(docBytes)).Error; err != nil {
			return err
		}
	}
	return nil
}

// nextCaseID returns the next case id (max + 1). CAST keeps it working even on
// legacy TEXT id columns that predate the uint64 autoincrement migration.
func (r *GormRepository) nextCaseID(ctx context.Context) (uint64, error) {
	var next uint64
	err := r.db.WithContext(ctx).
		Model(&CaseRow{}).
		Select("COALESCE(MAX(CAST(id AS INTEGER)), 0) + 1").
		Scan(&next).Error
	if err != nil {
		return 0, err
	}
	if next == 0 {
		return 1, nil
	}
	return next, nil
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
	err := r.db.WithContext(ctx).First(&row, "id = ?", id).Error
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
	if q.Tag != "" {
		tx = tx.Where("tags_json LIKE ?", "%\""+q.Tag+"\"%")
	}
	if q.Category != "" {
		tx = tx.Where("cats_json LIKE ?", "%\""+q.Category+"\"%")
	}
	if q.Q != "" {
		like := "%" + q.Q + "%"
		tx = tx.Where("id LIKE ? OR name LIKE ?", like, like)
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
	res := r.db.WithContext(ctx).Model(&CaseRow{}).Where("id = ?", id).Update("enabled", false)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *GormRepository) Enable(ctx context.Context, id sharedkernel.CaseID) error {
	res := r.db.WithContext(ctx).Model(&CaseRow{}).Where("id = ?", id).Update("enabled", true)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *GormRepository) Delete(ctx context.Context, id sharedkernel.CaseID) error {
	res := r.db.WithContext(ctx).Where("id = ?", id).Delete(&CaseRow{})
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
		ID:       uint64(c.Document.ID),
		Name:     c.Document.Name,
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
