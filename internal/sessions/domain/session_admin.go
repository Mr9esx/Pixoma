package domain

import (
	"context"


	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

type SessionAdminUser struct {
	ID             string
	ChannelID      string
	ExternalUserID string
	Username       string
	FirstName      string
	LastName       string
}

type SessionAdminContext struct {
	Session     *Session
	ChannelName string
	User        *SessionAdminUser
}

type SessionAdminReader interface {
	List(ctx context.Context, q ListQuery) ([]*SessionAdminContext, error)
	Get(ctx context.Context, id sharedkernel.SessionID) (*SessionAdminContext, error)
}
