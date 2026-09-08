package domain

import (
	"errors"
	"time"
)

const (
	RoleAdmin    = "admin"
	RoleOperator = "operator"
	RoleViewer   = "viewer"
)

var (
	ErrNotFound  = errors.New("console user: not found")
	ErrDuplicate = errors.New("console user: username or email already taken")
)

// ConsoleUser is a console login account, distinct from channel end-users.
type ConsoleUser struct {
	ID                 string
	Username           string
	Email              string
	Nickname           string
	AvatarURL          string
	Role               string
	Enabled            bool
	MustChangePassword bool
	PasswordHash       string
	LastLoginAt        time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
