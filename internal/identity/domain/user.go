package domain

import "time"

// User is the internal identity aggregate keyed by UUID, with channel-scoped
// external identities as the unique lookup key.
type User struct {
	ID           string
	Username     string
	FirstName    string
	LastName     string
	LanguageCode string
	LastSeenAt   time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
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
