package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	topicdomain "github.com/Mr9esx/Pixoma/internal/topics/domain"
)

// TopicRow is the GORM model for the topics table.
type TopicRow struct {
	Key       string    `gorm:"primaryKey;size:64"`
	Name      string    `gorm:"size:256;not null"`
	Enabled   bool      `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

func (TopicRow) TableName() string { return "topics" }

// TopicRepository is a GORM-backed topicdomain.Repository.
type TopicRepository struct {
	db *gorm.DB
}

// NewTopicRepository constructs a TopicRepository.
func NewTopicRepository(db *gorm.DB) *TopicRepository {
	return &TopicRepository{db: db}
}

func (r *TopicRepository) List(ctx context.Context, enabled *bool) ([]topicdomain.Topic, error) {
	q := r.db.WithContext(ctx)
	if enabled != nil {
		q = q.Where("enabled = ?", *enabled)
	}
	var rows []TopicRow
	if err := q.Order("key ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]topicdomain.Topic, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromRow(row))
	}
	return out, nil
}

func (r *TopicRepository) Get(ctx context.Context, key string) (*topicdomain.Topic, error) {
	var row TopicRow
	err := r.db.WithContext(ctx).First(&row, "key = ?", key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, topicdomain.ErrTopicNotFound
	}
	if err != nil {
		return nil, err
	}
	out := fromRow(row)
	return &out, nil
}

func (r *TopicRepository) Create(ctx context.Context, t topicdomain.Topic) error {
	if t.Key == "" || t.Name == "" {
		return fmt.Errorf("topic: key and name are required")
	}
	var n int64
	if err := r.db.WithContext(ctx).Model(&TopicRow{}).Where("key = ?", t.Key).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return topicdomain.ErrTopicConflict
	}
	return r.db.WithContext(ctx).Create(toRow(t)).Error
}

func (r *TopicRepository) Update(ctx context.Context, t topicdomain.Topic) error {
	if t.Key == "" {
		return fmt.Errorf("topic: key is required")
	}
	res := r.db.WithContext(ctx).Model(&TopicRow{}).Where("key = ?", t.Key).Updates(map[string]any{
		"name":       t.Name,
		"enabled":    t.Enabled,
		"updated_at": t.UpdatedAt,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return topicdomain.ErrTopicNotFound
	}
	return nil
}

func (r *TopicRepository) Delete(ctx context.Context, key string) error {
	res := r.db.WithContext(ctx).Where("key = ?", key).Delete(&TopicRow{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return topicdomain.ErrTopicNotFound
	}
	return nil
}

func toRow(t topicdomain.Topic) TopicRow {
	return TopicRow{
		Key:       t.Key,
		Name:      t.Name,
		Enabled:   t.Enabled,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

func fromRow(row TopicRow) topicdomain.Topic {
	return topicdomain.Topic{
		Key:       row.Key,
		Name:      row.Name,
		Enabled:   row.Enabled,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}
