package mcp

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

type TokenRow struct {
	UserID      string `gorm:"primaryKey;size:36"`
	TokenHash   string `gorm:"size:64;uniqueIndex;not null"`
	TokenCipher string `gorm:"type:text;not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (TokenRow) TableName() string { return "mcp_user_tokens" }

type GormTokenStore struct {
	db *gorm.DB
}

func NewGormTokenStore(db *gorm.DB) *GormTokenStore {
	return &GormTokenStore{db: db}
}

func (s *GormTokenStore) Put(ctx context.Context, rec TokenRecord) error {
	row := TokenRow{
		UserID: rec.UserID, TokenHash: rec.TokenHash, TokenCipher: rec.TokenCipher,
	}
	return s.db.WithContext(ctx).Save(&row).Error
}

func (s *GormTokenStore) GetByHash(ctx context.Context, hash string) (TokenRecord, error) {
	var row TokenRow
	err := s.db.WithContext(ctx).First(&row, "token_hash = ?", hash).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return TokenRecord{}, ErrTokenNotFound
	}
	if err != nil {
		return TokenRecord{}, err
	}
	return toTokenRecord(row), nil
}

func (s *GormTokenStore) GetByUserID(ctx context.Context, userID string) (TokenRecord, error) {
	var row TokenRow
	err := s.db.WithContext(ctx).First(&row, "user_id = ?", userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return TokenRecord{}, ErrTokenNotFound
	}
	if err != nil {
		return TokenRecord{}, err
	}
	return toTokenRecord(row), nil
}

func (s *GormTokenStore) DeleteByUserID(ctx context.Context, userID string) error {
	return s.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&TokenRow{}).Error
}

func toTokenRecord(row TokenRow) TokenRecord {
	return TokenRecord{UserID: row.UserID, TokenHash: row.TokenHash, TokenCipher: row.TokenCipher}
}
