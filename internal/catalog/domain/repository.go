package domain

import (
	"context"
	"errors"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

var (
	ErrNotFound      = errors.New("case not found")
	ErrAlreadyExists = errors.New("case already exists")
	ErrDisabled      = errors.New("case disabled")
)

// Case is the catalog aggregate: protocol document + availability.
type Case struct {
	Document CaseDocument
	Enabled  bool
}

type ListQuery struct {
	Tag         string
	Category    string
	Enabled     *bool // nil = all
	Q           string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	Limit       int
	Offset      int
}

type Repository interface {
	Save(ctx context.Context, c *Case) error
	// Create fails with ErrAlreadyExists if id is taken.
	Create(ctx context.Context, c *Case) error
	Get(ctx context.Context, id sharedkernel.CaseID) (*Case, error)
	List(ctx context.Context, q ListQuery) ([]*Case, error)
	Disable(ctx context.Context, id sharedkernel.CaseID) error
	Enable(ctx context.Context, id sharedkernel.CaseID) error
	// Delete removes a case row; ErrNotFound when the id does not exist.
	Delete(ctx context.Context, id sharedkernel.CaseID) error
}
