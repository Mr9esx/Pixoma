package persistence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/mr9esx/comfyui_tgbot/internal/identity/domain"
)

// UserRow is the GORM model for the channel_users table (internal identity).
type UserRow struct {
	ID           string            `gorm:"primaryKey;size:36"`
	Username     string            `gorm:"size:256"`
	FirstName    string            `gorm:"size:256"`
	LastName     string            `gorm:"size:256"`
	LanguageCode string            `gorm:"size:64"`
	Access       domain.UserAccess `gorm:"size:32;not null;default:'denied'"`
	LastSeenAt   time.Time         `gorm:"not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (UserRow) TableName() string { return "channel_users" }

// UserExternalIdentityRow maps an internal user to a channel-scoped external id.
type UserExternalIdentityRow struct {
	ID             string    `gorm:"primaryKey;size:36"`
	UserID         string    `gorm:"size:36;not null;index"`
	ChannelID      string    `gorm:"size:128;not null;uniqueIndex:idx_channel_external"`
	ExternalUserID string    `gorm:"size:256;not null;uniqueIndex:idx_channel_external"`
	ProfileJSON    string    `gorm:"type:text"`
	LastSeenAt     time.Time `gorm:"not null"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (UserExternalIdentityRow) TableName() string { return "channel_user_external_identities" }

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

// UpsertByChannelExternal upserts an internal user keyed by (channel, external id).
func (r *UserRepository) UpsertByChannelExternal(ctx context.Context, in domain.UpsertFrom) (*domain.User, error) {
	if in.ChannelID == "" || in.ExternalUserID == "" {
		return nil, fmt.Errorf("identity: channel_id and external_user_id required")
	}
	for attempt := 0; attempt < 3; attempt++ {
		got, err := r.upsertOnce(ctx, in)
		if err == nil || !isUniqueViolation(err) {
			return got, err
		}
		// 并发创建同一条 (channel, external_user_id) 时输掉竞态：重读胜出行后走更新路径。
	}
	return nil, fmt.Errorf("identity: concurrent upsert retries exhausted (%s, %s)", in.ChannelID, in.ExternalUserID)
}

func (r *UserRepository) upsertOnce(ctx context.Context, in domain.UpsertFrom) (*domain.User, error) {
	now := in.LastSeenAt
	if now.IsZero() {
		now = r.now()
	}

	var identity UserExternalIdentityRow
	err := r.db.WithContext(ctx).
		Where("channel_id = ? AND external_user_id = ?", in.ChannelID, in.ExternalUserID).
		Limit(1).
		Find(&identity).Error
	if err != nil {
		return nil, err
	}

	if identity.ID == "" {
		userID := uuid.NewString()
		user := UserRow{
			ID: userID, Username: in.Username, FirstName: in.FirstName,
			LastName: in.LastName, LanguageCode: in.LanguageCode,
			Access: domain.UserAccessDenied, LastSeenAt: now,
			CreatedAt: now, UpdatedAt: now,
		}
		ident := UserExternalIdentityRow{
			ID: uuid.NewString(), UserID: userID,
			ChannelID: in.ChannelID, ExternalUserID: in.ExternalUserID,
			ProfileJSON: in.ProfileJSON, LastSeenAt: now, CreatedAt: now, UpdatedAt: now,
		}
		return fromRow(user), r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&user).Error; err != nil {
				return err
			}
			return tx.Create(&ident).Error
		})
	}

	var user UserRow
	if err := r.db.WithContext(ctx).First(&user, "id = ?", identity.UserID).Error; err != nil {
		return nil, err
	}
	user.Username = in.Username
	user.FirstName = in.FirstName
	user.LastName = in.LastName
	user.LanguageCode = in.LanguageCode
	user.LastSeenAt = now
	user.UpdatedAt = now
	identity.ProfileJSON = in.ProfileJSON
	identity.LastSeenAt = now
	identity.UpdatedAt = now
	return fromRow(user), r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&user).Error; err != nil {
			return err
		}
		return tx.Save(&identity).Error
	})
}

// isUniqueViolation reports whether err is a unique-index conflict across
// SQLite / MySQL / PostgreSQL drivers.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") || // sqlite
		strings.Contains(msg, "duplicate entry") || // mysql
		strings.Contains(msg, "duplicate key") // postgres
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
	user := fromRow(row)
	if err := r.attachIdentities(ctx, []*domain.User{user}); err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) SetAccess(ctx context.Context, id string, access domain.UserAccess) (*domain.User, error) {
	access = domain.NormalizeUserAccess(string(access))
	var row UserRow
	err := r.db.WithContext(ctx).First(&row, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	row.Access = access
	row.UpdatedAt = r.now()
	if err := r.db.WithContext(ctx).Save(&row).Error; err != nil {
		return nil, err
	}
	user := fromRow(row)
	if err := r.attachIdentities(ctx, []*domain.User{user}); err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) List(ctx context.Context, q domain.ListQuery) ([]*domain.User, error) {
	tx := r.db.WithContext(ctx).Model(&UserRow{})
	if q.ChannelID != nil || q.ExternalUserID != nil {
		sub := r.db.WithContext(ctx).Model(&UserExternalIdentityRow{}).Select("user_id")
		if q.ChannelID != nil {
			sub = sub.Where("channel_id = ?", *q.ChannelID)
		}
		if q.ExternalUserID != nil {
			sub = sub.Where("external_user_id = ?", *q.ExternalUserID)
		}
		tx = tx.Where("id IN (?)", sub)
	}
	if q.Q != "" {
		like := "%" + q.Q + "%"
		tx = tx.Where(
			"id LIKE ? OR username LIKE ? OR first_name LIKE ? OR last_name LIKE ? OR EXISTS ("+
				"SELECT 1 FROM channel_user_external_identities WHERE user_id = channel_users.id AND external_user_id LIKE ?"+
				")",
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
	if err := r.attachIdentities(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *UserRepository) attachIdentities(ctx context.Context, users []*domain.User) error {
	if len(users) == 0 {
		return nil
	}
	ids := make([]string, 0, len(users))
	byID := make(map[string]*domain.User, len(users))
	for _, user := range users {
		ids = append(ids, user.ID)
		byID[user.ID] = user
	}
	var identities []UserExternalIdentityRow
	if err := r.db.WithContext(ctx).
		Where("user_id IN ?", ids).
		Order("created_at, id").
		Find(&identities).Error; err != nil {
		return err
	}
	for _, identity := range identities {
		if user, ok := byID[identity.UserID]; ok && user.ChannelID == "" {
			user.ChannelID = identity.ChannelID
			user.ExternalUserID = identity.ExternalUserID
		}
	}
	return nil
}

func fromRow(row UserRow) *domain.User {
	return &domain.User{
		ID:           row.ID,
		Username:     row.Username,
		FirstName:    row.FirstName,
		LastName:     row.LastName,
		LanguageCode: row.LanguageCode,
		Access:       domain.NormalizeUserAccess(string(row.Access)),
		LastSeenAt:   row.LastSeenAt,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}
