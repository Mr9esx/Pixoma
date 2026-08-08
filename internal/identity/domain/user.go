package domain

import "time"

// User is the internal identity aggregate keyed by UUID, with Telegram id as unique channel key.
type User struct {
	ID           string
	TgUserID     int64
	Username     string
	FirstName    string
	LastName     string
	LanguageCode string
	IsBot        *bool
	IsPremium    *bool
	LastSeenAt   time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// UpsertFrom holds Telegram From fields available for upsert.
type UpsertFrom struct {
	TgUserID     int64
	Username     string
	FirstName    string
	LastName     string
	LanguageCode string
	IsBot        *bool
	IsPremium    *bool
}
