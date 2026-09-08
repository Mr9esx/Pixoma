package domain

import (
	"strings"
	"time"
)

type UserAccess string

const (
	UserAccessAlwaysAllowed UserAccess = "always_allowed"
	UserAccessPaid          UserAccess = "paid"
	UserAccessDenied        UserAccess = "denied"
)

func NormalizeUserAccess(value string) UserAccess {
	switch UserAccess(strings.ToLower(strings.TrimSpace(value))) {
	case UserAccessAlwaysAllowed:
		return UserAccessAlwaysAllowed
	case UserAccessPaid:
		return UserAccessPaid
	default:
		return UserAccessDenied
	}
}

// User is the internal identity aggregate keyed by UUID, with channel-scoped
// external identities as the unique lookup key.
type User struct {
	ID             string
	ChannelID      string
	ChannelName    string
	ExternalUserID string
	Username       string
	FirstName      string
	LastName       string
	LanguageCode   string
	Access         UserAccess
	LastSeenAt     time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// UpsertFrom carries channel-scoped external identity and profile fields.
type UpsertFrom struct {
	ChannelID      string
	ExternalUserID string
	Username       string
	FirstName      string
	LastName       string
	LanguageCode   string
	ProfileJSON    string // platform-specific profile fields (e.g. is_bot/is_premium)
	LastSeenAt     time.Time
}
