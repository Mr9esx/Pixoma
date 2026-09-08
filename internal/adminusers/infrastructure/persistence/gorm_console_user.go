package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	consoledomain "github.com/Mr9esx/Pixoma/internal/adminusers/domain"
)

// ConsoleUserRow is the GORM model for the console_users table.
type ConsoleUserRow struct {
	ID                 string    `gorm:"primaryKey;size:36"`
	Username           string    `gorm:"size:128;not null;uniqueIndex:idx_console_username"`
	Email              *string   `gorm:"size:256;uniqueIndex:idx_console_email"`
	Nickname           string    `gorm:"size:256"`
	AvatarURL          string    `gorm:"size:512"`
	Role               string    `gorm:"size:32;not null"`
	Enabled            bool      `gorm:"not null"`
	MustChangePassword bool      `gorm:"not null"`
	PasswordHash       string    `gorm:"type:text;not null"`
	LastLoginAt        time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (ConsoleUserRow) TableName() string { return "console_users" }

func fromRow(row ConsoleUserRow) *consoledomain.ConsoleUser {
	email := ""
	if row.Email != nil {
		email = *row.Email
	}
	return &consoledomain.ConsoleUser{
		ID:                 row.ID,
		Username:           row.Username,
		Email:              email,
		Nickname:           row.Nickname,
		AvatarURL:          row.AvatarURL,
		Role:               row.Role,
		Enabled:            row.Enabled,
		MustChangePassword: row.MustChangePassword,
		PasswordHash:       row.PasswordHash,
		LastLoginAt:        row.LastLoginAt,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}

func toRow(u *consoledomain.ConsoleUser) ConsoleUserRow {
	var email *string
	if e := strings.TrimSpace(u.Email); e != "" {
		email = &e
	}
	return ConsoleUserRow{
		ID:                 u.ID,
		Username:           u.Username,
		Email:              email,
		Nickname:           u.Nickname,
		AvatarURL:          u.AvatarURL,
		Role:               u.Role,
		Enabled:            u.Enabled,
		MustChangePassword: u.MustChangePassword,
		PasswordHash:       u.PasswordHash,
		LastLoginAt:        u.LastLoginAt,
	}
}

// ConsoleUserRepository is a GORM-backed consoledomain.Repository.
type ConsoleUserRepository struct {
	db *gorm.DB
}

func NewConsoleUserRepository(db *gorm.DB) *ConsoleUserRepository {
	return &ConsoleUserRepository{db: db}
}

func (r *ConsoleUserRepository) Create(ctx context.Context, u *consoledomain.ConsoleUser) error {
	if u == nil {
		return errors.New("console user: nil user")
	}
	if strings.TrimSpace(u.Username) == "" {
		return errors.New("console user: username required")
	}
	exists, err := r.usernameExists(ctx, u.Username)
	if err != nil {
		return err
	}
	if exists {
		return consoledomain.ErrDuplicate
	}
	if strings.TrimSpace(u.Email) != "" {
		exists, err = r.emailExists(ctx, u.Email)
		if err != nil {
			return err
		}
		if exists {
			return consoledomain.ErrDuplicate
		}
	}
	row := toRow(u)
	return r.db.WithContext(ctx).Create(&row).Error
}

func (r *ConsoleUserRepository) GetByUsername(ctx context.Context, username string) (*consoledomain.ConsoleUser, error) {
	var row ConsoleUserRow
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, consoledomain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return fromRow(row), nil
}

func (r *ConsoleUserRepository) GetByID(ctx context.Context, id string) (*consoledomain.ConsoleUser, error) {
	var row ConsoleUserRow
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, consoledomain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return fromRow(row), nil
}

func (r *ConsoleUserRepository) List(ctx context.Context, q consoledomain.ListQuery) ([]*consoledomain.ConsoleUser, error) {
	query := r.db.WithContext(ctx).Model(&ConsoleUserRow{})
	if s := strings.TrimSpace(q.Q); s != "" {
		like := "%" + s + "%"
		query = query.Where("username LIKE ? OR email LIKE ? OR nickname LIKE ?", like, like, like)
	}
	if q.Limit > 0 {
		query = query.Limit(q.Limit)
	}
	if q.Offset > 0 {
		query = query.Offset(q.Offset)
	}
	var rows []ConsoleUserRow
	if err := query.Order("created_at ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*consoledomain.ConsoleUser, 0, len(rows))
	for i := range rows {
		out = append(out, fromRow(rows[i]))
	}
	return out, nil
}

func (r *ConsoleUserRepository) CountAdmins(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&ConsoleUserRow{}).
		Where("role = ? AND enabled = ?", consoledomain.RoleAdmin, true).
		Count(&n).Error
	return n, err
}

func (r *ConsoleUserRepository) Update(ctx context.Context, u *consoledomain.ConsoleUser) error {
	if u == nil {
		return errors.New("console user: nil user")
	}
	return r.db.WithContext(ctx).Model(&ConsoleUserRow{}).
		Where("id = ?", u.ID).
		Updates(map[string]any{
			"email":                emailForUpdate(u.Email),
			"nickname":             u.Nickname,
			"avatar_url":           u.AvatarURL,
			"role":                 u.Role,
			"enabled":              u.Enabled,
			"must_change_password": u.MustChangePassword,
			"password_hash":        u.PasswordHash,
			"last_login_at":        u.LastLoginAt,
		}).Error
}

func (r *ConsoleUserRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&ConsoleUserRow{}).Error
}

func (r *ConsoleUserRepository) usernameExists(ctx context.Context, username string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&ConsoleUserRow{}).
		Where("username = ?", username).Count(&n).Error
	return n > 0, err
}

func (r *ConsoleUserRepository) emailExists(ctx context.Context, email string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&ConsoleUserRow{}).
		Where("email = ?", email).Count(&n).Error
	return n > 0, err
}

// emailForUpdate returns a SQL NULL when email is empty so the unique index
// permits multiple users with no email.
func emailForUpdate(email string) *string {
	if e := strings.TrimSpace(email); e != "" {
		return &e
	}
	return nil
}
