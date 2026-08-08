package domain

import (
	"errors"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

var (
	ErrSessionLocked   = errors.New("session locked")
	ErrNoActiveSession = errors.New("no active session")
	ErrInvalidState    = errors.New("invalid session state")
	ErrInputOutOfOrder = errors.New("input out of order")
)

type Status string

const (
	StatusCollecting Status = "collecting"
	StatusConfirming Status = "confirming"
	StatusSubmitted  Status = "submitted"
	StatusExited     Status = "exited"
)

func (s Status) IsActive() bool {
	return s == StatusCollecting || s == StatusConfirming
}

// DraftValue is a collected input before ConfirmRun.
type DraftValue struct {
	Key    string
	Text   *string
	Number *float64
	Bool   *bool
	Blob   *sharedkernel.BlobRef
	Skipped bool
}

type Session struct {
	ID                sharedkernel.SessionID
	UserID            string
	ChatID            sharedkernel.ChatID
	CaseID            sharedkernel.CaseID
	Status            Status
	CurrentInputIndex int
	InputKeys         []string // ordered keys from case inputs
	Draft             map[string]DraftValue
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func NewCollecting(id sharedkernel.SessionID, chat sharedkernel.ChatID, caseID sharedkernel.CaseID, inputKeys []string, now time.Time) *Session {
	return &Session{
		ID:        id,
		ChatID:    chat,
		CaseID:    caseID,
		Status:    StatusCollecting,
		InputKeys: append([]string(nil), inputKeys...),
		Draft:     map[string]DraftValue{},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (s *Session) currentKey() (string, bool) {
	if s.CurrentInputIndex < 0 || s.CurrentInputIndex >= len(s.InputKeys) {
		return "", false
	}
	return s.InputKeys[s.CurrentInputIndex], true
}

func (s *Session) SubmitInput(v DraftValue, now time.Time) error {
	if s.Status != StatusCollecting {
		return ErrInvalidState
	}
	key, ok := s.currentKey()
	if !ok {
		return ErrInvalidState
	}
	if v.Key != "" && v.Key != key {
		return ErrInputOutOfOrder
	}
	v.Key = key
	s.Draft[key] = v
	s.CurrentInputIndex++
	s.UpdatedAt = now
	if s.CurrentInputIndex >= len(s.InputKeys) {
		s.Status = StatusConfirming
	}
	return nil
}

func (s *Session) SkipInput(now time.Time) error {
	if s.Status != StatusCollecting {
		return ErrInvalidState
	}
	key, ok := s.currentKey()
	if !ok {
		return ErrInvalidState
	}
	s.Draft[key] = DraftValue{Key: key, Skipped: true}
	s.CurrentInputIndex++
	s.UpdatedAt = now
	if s.CurrentInputIndex >= len(s.InputKeys) {
		s.Status = StatusConfirming
	}
	return nil
}

func (s *Session) Exit(now time.Time) error {
	if !s.Status.IsActive() {
		return ErrInvalidState
	}
	s.Status = StatusExited
	s.UpdatedAt = now
	return nil
}

func (s *Session) MarkSubmitted(now time.Time) error {
	if s.Status != StatusConfirming {
		return ErrInvalidState
	}
	s.Status = StatusSubmitted
	s.UpdatedAt = now
	return nil
}

func (s *Session) BackToCollecting(now time.Time) error {
	if s.Status != StatusConfirming {
		return ErrInvalidState
	}
	s.Status = StatusCollecting
	if s.CurrentInputIndex > 0 {
		s.CurrentInputIndex = len(s.InputKeys) - 1
		if s.CurrentInputIndex < 0 {
			s.CurrentInputIndex = 0
		}
	}
	s.UpdatedAt = now
	return nil
}
