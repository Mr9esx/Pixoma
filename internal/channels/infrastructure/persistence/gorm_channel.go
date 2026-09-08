package persistence

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/Mr9esx/Pixoma/internal/channels/domain"
)

// ChannelRow is the GORM model for the channels table.
type ChannelRow struct {
	ID                   string `gorm:"primaryKey;size:128"`
	Platform             string `gorm:"size:32;not null"`
	Name                 string `gorm:"size:256;not null"`
	ExtraInfo            string `gorm:"type:text"`
	CredentialCiphertext string `gorm:"type:text;not null"`
	Enabled              bool   `gorm:"not null;default:true"`
	LastCheckKind        string `gorm:"size:32"`
	LastCheckMessage     string `gorm:"type:text"`
	LastCheckAt          *time.Time
	CreatedAt            time.Time `gorm:"not null"`
	UpdatedAt            time.Time `gorm:"not null"`
}

func (ChannelRow) TableName() string { return "channels" }

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) Create(ctx context.Context, ch domain.Channel) error {
	row := rowFromDomain(ch)
	return r.db.WithContext(ctx).Create(&row).Error
}

func (r *GormRepository) Get(ctx context.Context, id string) (domain.Channel, error) {
	var row ChannelRow
	err := r.db.WithContext(ctx).First(&row, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.Channel{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Channel{}, err
	}
	return row.toDomain(), nil
}

func (r *GormRepository) List(ctx context.Context) ([]domain.Channel, error) {
	var rows []ChannelRow
	if err := r.db.WithContext(ctx).Order("created_at, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Channel, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.toDomain())
	}
	return out, nil
}

func (r *GormRepository) ChannelNames(ctx context.Context, ids []string) (map[string]string, error) {
	if len(ids) == 0 {
		return map[string]string{}, nil
	}
	var rows []ChannelRow
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	names := make(map[string]string, len(rows))
	for _, row := range rows {
		names[row.ID] = row.Name
	}
	return names, nil
}

func (r *GormRepository) Update(ctx context.Context, ch domain.Channel) error {
	row := rowFromDomain(ch)
	return r.db.WithContext(ctx).Save(&row).Error
}

func (r *GormRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&ChannelRow{}, "id = ?", id).Error
}

func rowFromDomain(ch domain.Channel) ChannelRow {
	return ChannelRow{
		ID:                   ch.ID,
		Platform:             ch.Platform,
		Name:                 ch.Name,
		ExtraInfo:            ch.ExtraInfo,
		CredentialCiphertext: ch.CredentialCiphertext,
		Enabled:              ch.Enabled,
		LastCheckKind:        ch.LastCheckKind,
		LastCheckMessage:     ch.LastCheckMessage,
		LastCheckAt:          ch.LastCheckAt,
		CreatedAt:            ch.CreatedAt,
		UpdatedAt:            ch.UpdatedAt,
	}
}

func (r ChannelRow) toDomain() domain.Channel {
	return domain.Channel{
		ID:                   r.ID,
		Platform:             r.Platform,
		Name:                 r.Name,
		ExtraInfo:            r.ExtraInfo,
		CredentialCiphertext: r.CredentialCiphertext,
		Enabled:              r.Enabled,
		LastCheckKind:        r.LastCheckKind,
		LastCheckMessage:     r.LastCheckMessage,
		LastCheckAt:          r.LastCheckAt,
		CreatedAt:            r.CreatedAt,
		UpdatedAt:            r.UpdatedAt,
	}
}
