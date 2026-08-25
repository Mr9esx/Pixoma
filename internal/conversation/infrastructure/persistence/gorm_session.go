package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// SessionRow is the GORM model for the sessions table.
type SessionRow struct {
	ID                string    `gorm:"primaryKey;size:36"`
	UserID            string    `gorm:"size:36;index;not null"`
	ChannelID         string    `gorm:"size:128;not null;default:'';index:idx_chat_active"`
	ChatExternalID    string    `gorm:"size:256;not null;default:'';index:idx_chat_active"`
	Status            string    `gorm:"size:32;not null;index:idx_chat_active"`
	CaseID            uint64    `gorm:"not null"`
	CurrentInputIndex int       `gorm:"not null;default:0"`
	InputKeysJSON     string    `gorm:"column:input_keys_json;type:text;not null"`
	DraftJSON         string    `gorm:"column:draft_json;type:text;not null"`
	CreatedAt         time.Time `gorm:"not null"`
	UpdatedAt         time.Time `gorm:"not null"`
}

func (SessionRow) TableName() string { return "sessions" }

// SessionRepository is a GORM-backed conversation.Repository.
type SessionRepository struct {
	db *gorm.DB
}

// NewSessionRepository constructs a SessionRepository.
func NewSessionRepository(db *gorm.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) GetActiveByChat(ctx context.Context, chatID sharedkernel.ChatID) (*domain.Session, error) {
	addr, err := sharedkernel.ParseChatID(string(chatID))
	if err != nil {
		return nil, err
	}
	var row SessionRow
	err = r.db.WithContext(ctx).
		Where("channel_id = ? AND chat_external_id = ? AND status IN ?", addr.ChannelID, addr.ExternalChatID, []string{
			string(domain.StatusCollecting),
			string(domain.StatusConfirming),
		}).
		Limit(1).
		Find(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == "" {
		return nil, domain.ErrNoActiveSession
	}
	return fromRow(row)
}

func (r *SessionRepository) GetByID(ctx context.Context, id sharedkernel.SessionID) (*domain.Session, error) {
	var row SessionRow
	err := r.db.WithContext(ctx).First(&row, "id = ?", string(id)).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return fromRow(row)
}

func (r *SessionRepository) List(ctx context.Context, q domain.ListQuery) ([]*domain.Session, error) {
	tx := r.db.WithContext(ctx).Model(&SessionRow{})
	if q.UserID != "" {
		tx = tx.Where("user_id = ?", q.UserID)
	}
	if q.ChatID != nil {
		addr, err := sharedkernel.ParseChatID(string(*q.ChatID))
		if err != nil {
			return nil, err
		}
		tx = tx.Where("channel_id = ? AND chat_external_id = ?", addr.ChannelID, addr.ExternalChatID)
	}
	if q.Status != "" {
		tx = tx.Where("status = ?", string(q.Status))
	}
	if q.CaseID != 0 {
		tx = tx.Where("case_id = ?", uint64(q.CaseID))
	}
	if q.ChannelID != "" {
		tx = tx.Where("channel_id = ?", q.ChannelID)
	}
	if q.Q != "" {
		like := "%" + q.Q + "%"
		tx = tx.Where("id LIKE ? OR case_id LIKE ? OR user_id LIKE ?", like, like, like)
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
	var rows []SessionRow
	if err := tx.Order("updated_at desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.Session, 0, len(rows))
	for _, row := range rows {
		s, err := fromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func (r *SessionRepository) ListActiveByCase(ctx context.Context, caseID sharedkernel.CaseID) ([]*domain.Session, error) {
	var rows []SessionRow
	if err := r.db.WithContext(ctx).
		Where("case_id = ? AND status IN ?", uint64(caseID), []string{
			string(domain.StatusCollecting),
			string(domain.StatusConfirming),
		}).
		Order("updated_at desc").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.Session, 0, len(rows))
	for _, row := range rows {
		s, err := fromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func (r *SessionRepository) Save(ctx context.Context, s *domain.Session) error {
	if s == nil {
		return fmt.Errorf("conversation: nil session")
	}
	row, err := toRow(s)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(row).Error
}

func (r *SessionRepository) ClearActive(ctx context.Context, chatID sharedkernel.ChatID) error {
	addr, err := sharedkernel.ParseChatID(string(chatID))
	if err != nil {
		return err
	}
	// Do not physically delete; mark active rows exited so Task can still join by id.
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&SessionRow{}).
		Where("channel_id = ? AND chat_external_id = ? AND status IN ?", addr.ChannelID, addr.ExternalChatID, []string{
			string(domain.StatusCollecting),
			string(domain.StatusConfirming),
		}).
		Updates(map[string]any{
			"status":     string(domain.StatusExited),
			"updated_at": now,
		}).Error
}

// CountByChannel counts active sessions for a channel (delete-restriction check).
func (r *SessionRepository) CountByChannel(ctx context.Context, channelID string) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&SessionRow{}).
		Where("channel_id = ? AND status IN ?", channelID, []string{
			string(domain.StatusCollecting),
			string(domain.StatusConfirming),
		}).
		Count(&n).Error
	return n, err
}

func toRow(s *domain.Session) (*SessionRow, error) {
	keys, err := json.Marshal(s.InputKeys)
	if err != nil {
		return nil, err
	}
	draft := s.Draft
	if draft == nil {
		draft = map[string]domain.DraftValue{}
	}
	draftBytes, err := json.Marshal(draft)
	if err != nil {
		return nil, err
	}
	addr, err := sharedkernel.ParseChatID(string(s.ChatID))
	if err != nil {
		return nil, err
	}
	return &SessionRow{
		ID:                string(s.ID),
		UserID:            s.UserID,
		ChannelID:         addr.ChannelID,
		ChatExternalID:    addr.ExternalChatID,
		CaseID:            uint64(s.CaseID),
		Status:            string(s.Status),
		CurrentInputIndex: s.CurrentInputIndex,
		InputKeysJSON:     string(keys),
		DraftJSON:         string(draftBytes),
		CreatedAt:         s.CreatedAt,
		UpdatedAt:         s.UpdatedAt,
	}, nil
}

func fromRow(row SessionRow) (*domain.Session, error) {
	var keys []string
	if row.InputKeysJSON != "" {
		if err := json.Unmarshal([]byte(row.InputKeysJSON), &keys); err != nil {
			return nil, fmt.Errorf("conversation: decode input_keys: %w", err)
		}
	}
	draft := map[string]domain.DraftValue{}
	if row.DraftJSON != "" {
		if err := json.Unmarshal([]byte(row.DraftJSON), &draft); err != nil {
			return nil, fmt.Errorf("conversation: decode draft: %w", err)
		}
	}
	return &domain.Session{
		ID:                sharedkernel.SessionID(row.ID),
		UserID:            row.UserID,
		ChatID:            sharedkernel.ChatID(sharedkernel.FormatChatID(sharedkernel.ChannelAddr{ChannelID: row.ChannelID, ExternalChatID: row.ChatExternalID})),
		CaseID:            sharedkernel.CaseID(row.CaseID),
		Status:            domain.Status(row.Status),
		CurrentInputIndex: row.CurrentInputIndex,
		InputKeys:         keys,
		Draft:             draft,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}, nil
}
