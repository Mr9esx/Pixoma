package persistence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type BlobWriteIntentRow struct {
	KeyHash    string `gorm:"primaryKey;size:64"`
	BlobKey    string `gorm:"type:text;not null"`
	AccountID  string `gorm:"size:64;not null;index"`
	CreatedAt  time.Time
	ExpiresAt  time.Time  `gorm:"index:idx_studio_blob_intent_queue,priority:1"`
	LeaseUntil *time.Time `gorm:"index:idx_studio_blob_intent_queue,priority:2"`
}

func (BlobWriteIntentRow) TableName() string { return "studio_blob_write_intents" }

func blobIntentKeyHash(accountID, blobKey string) string {
	sum := sha256.Sum256([]byte(accountID + "\x00" + blobKey))
	return hex.EncodeToString(sum[:])
}

func (r *GormRepository) RegisterBlobWriteIntent(ctx context.Context, intent *domain.BlobWriteIntent) error {
	if intent == nil || intent.BlobKey == "" || intent.AccountID == "" || !intent.ExpiresAt.After(intent.CreatedAt) {
		return fmt.Errorf("%w: invalid blob write intent", domain.ErrInvalid)
	}
	return translateCreateError(r.db.WithContext(ctx).Create(&BlobWriteIntentRow{KeyHash: blobIntentKeyHash(intent.AccountID, intent.BlobKey), BlobKey: intent.BlobKey, AccountID: intent.AccountID, CreatedAt: intent.CreatedAt.UTC(), ExpiresAt: intent.ExpiresAt.UTC()}).Error)
}

func (r *GormRepository) ClaimExpiredBlobWriteIntents(ctx context.Context, limit int, now time.Time) ([]*domain.BlobWriteIntent, error) {
	limit = normalizeLimit(limit)
	now = now.UTC()
	var rows []BlobWriteIntentRow
	if err := r.db.WithContext(ctx).Where("expires_at <= ? AND (lease_until IS NULL OR lease_until <= ?)", now, now).
		Order("expires_at ASC, blob_key ASC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	lease := now.Add(time.Minute)
	out := make([]*domain.BlobWriteIntent, 0, len(rows))
	for _, row := range rows {
		result := r.db.WithContext(ctx).Model(&BlobWriteIntentRow{}).
			Where("key_hash = ? AND account_id = ? AND expires_at <= ? AND (lease_until IS NULL OR lease_until <= ?)", row.KeyHash, row.AccountID, now, now).
			Update("lease_until", lease)
		if result.Error != nil {
			return nil, result.Error
		}
		if result.RowsAffected == 0 {
			continue
		}
		out = append(out, &domain.BlobWriteIntent{BlobKey: row.BlobKey, AccountID: row.AccountID, CreatedAt: row.CreatedAt, ExpiresAt: row.ExpiresAt, LeaseUntil: &lease})
	}
	return out, nil
}

func (r *GormRepository) IsBlobReferenced(ctx context.Context, accountID, blobKey string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&AssetVersionRow{}).Where("account_id = ? AND blob_key = ?", accountID, blobKey).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *GormRepository) DeleteBlobWriteIntent(ctx context.Context, accountID, blobKey string) error {
	if accountID == "" || blobKey == "" {
		return fmt.Errorf("%w: invalid blob write intent", domain.ErrInvalid)
	}
	return r.db.WithContext(ctx).Where("key_hash = ? AND account_id = ?", blobIntentKeyHash(accountID, blobKey), accountID).Delete(&BlobWriteIntentRow{}).Error
}
