package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/mr9esx/comfyui_tgbot/internal/identity/domain"
)

// UserRow is the GORM model for the users table.
type UserRow struct {
	ID           string `gorm:"primaryKey;size:36"`
	TgUserID     int64  `gorm:"uniqueIndex;not null"`
	Username     string `gorm:"size:256"`
	FirstName    string `gorm:"size:256"`
	LastName     string `gorm:"size:256"`
	LanguageCode string `gorm:"size:64"`
	IsBot        *bool
	IsPremium    *bool
	LastSeenAt   time.Time `gorm:"not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (UserRow) TableName() string { return "users" }

// UserRepository is a GORM-backed identity.Repository.
type UserRepository struct {
	db  *gorm.DB
	now func() time.Time
}

// NewUserRepository constructs a UserRepository.
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		db:  db,
		now: time.Now,
	}
}

func (r *UserRepository) UpsertByTgUserID(ctx context.Context, in domain.UpsertFrom) (*domain.User, error) {
	if in.TgUserID == 0 {
		return nil, fmt.Errorf("identity: empty tg_user_id")
	}
	now := r.now()
	var row UserRow
	err := r.db.WithContext(ctx).Where("tg_user_id = ?", in.TgUserID).Limit(1).Find(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == "" {
		row = UserRow{
			ID:           uuid.NewString(),
			TgUserID:     in.TgUserID,
			Username:     in.Username,
			FirstName:    in.FirstName,
			LastName:     in.LastName,
			LanguageCode: in.LanguageCode,
			IsBot:        in.IsBot,
			IsPremium:    in.IsPremium,
			LastSeenAt:   now,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
			return nil, err
		}
		return fromRow(row), nil
	}

	row.Username = in.Username
	row.FirstName = in.FirstName
	row.LastName = in.LastName
	row.LanguageCode = in.LanguageCode
	row.IsBot = in.IsBot
	row.IsPremium = in.IsPremium
	row.LastSeenAt = now
	row.UpdatedAt = now
	if err := r.db.WithContext(ctx).Save(&row).Error; err != nil {
		return nil, err
	}
	return fromRow(row), nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	var row UserRow
	err := r.db.WithContext(ctx).First(&row, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return fromRow(row), nil
}

func (r *UserRepository) List(ctx context.Context, q domain.ListQuery) ([]*domain.User, error) {
	tx := r.db.WithContext(ctx).Model(&UserRow{})
	if q.TgUserID != nil {
		tx = tx.Where("tg_user_id = ?", *q.TgUserID)
	}
	if q.Q != "" {
		like := "%" + q.Q + "%"
		tx = tx.Where(
			"id LIKE ? OR username LIKE ? OR first_name LIKE ? OR last_name LIKE ? OR CAST(tg_user_id AS TEXT) LIKE ?",
			like, like, like, like, like,
		)
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
	var rows []UserRow
	if err := tx.Order("created_at desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.User, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromRow(row))
	}
	return out, nil
}

func fromRow(row UserRow) *domain.User {
	return &domain.User{
		ID:           row.ID,
		TgUserID:     row.TgUserID,
		Username:     row.Username,
		FirstName:    row.FirstName,
		LastName:     row.LastName,
		LanguageCode: row.LanguageCode,
		IsBot:        row.IsBot,
		IsPremium:    row.IsPremium,
		LastSeenAt:   row.LastSeenAt,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}
