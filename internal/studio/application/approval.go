package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type ApprovalRepository interface {
	GetApproval(ctx context.Context, accountID, approvalID string) (*domain.Approval, error)
	UpdateApproval(ctx context.Context, approval *domain.Approval) error
	GetRun(ctx context.Context, accountID, runID string) (*domain.Run, error)
	UpdateRun(ctx context.Context, run *domain.Run) error
	AppendEvent(ctx context.Context, event *domain.Event) error
}

type ApprovalService struct {
	Repo  ApprovalRepository
	Queue RunQueue
	IDs   func() string
	Now   func() time.Time
}

type ResolveApprovalInput struct {
	AccountID  string
	ApprovalID string
	Approved   bool
}

func (s *ApprovalService) Resolve(ctx context.Context, input ResolveApprovalInput) error {
	if s == nil || s.Repo == nil || s.Queue == nil {
		return fmt.Errorf("studio: approval service is not configured")
	}
	approval, err := s.Repo.GetApproval(ctx, input.AccountID, input.ApprovalID)
	if err != nil {
		return err
	}
	now := s.now()
	if input.Approved {
		err = approval.Approve(input.AccountID, now)
	} else {
		err = approval.Reject(input.AccountID, now)
	}
	if err != nil {
		return err
	}
	if err := s.Repo.UpdateApproval(ctx, approval); err != nil {
		return err
	}
	run, err := s.Repo.GetRun(ctx, input.AccountID, approval.RunID)
	if err != nil {
		return err
	}
	if input.Approved {
		if err := run.Resume(now); err != nil {
			return err
		}
	} else if err := run.Cancel(now); err != nil {
		return err
	}
	if err := s.Repo.UpdateRun(ctx, run); err != nil {
		return err
	}
	events, err := s.nextEventSequence(ctx, input.AccountID, run.ID)
	if err != nil {
		return err
	}
	status := "rejected"
	if input.Approved {
		status = "approved"
	}
	payload := []byte(fmt.Sprintf(`{"approval_id":%q,"status":%q}`, approval.ID, status))
	if err := s.Repo.AppendEvent(ctx, &domain.Event{
		ID: s.nextID(), RunID: run.ID, SessionID: run.SessionID, AccountID: run.AccountID,
		Sequence: events, Type: EventApprovalResolved, Payload: payload, CreatedAt: now,
	}); err != nil {
		return err
	}
	if !input.Approved {
		return nil
	}
	return s.Queue.Enqueue(RunRef{AccountID: input.AccountID, RunID: run.ID})
}

func (s *ApprovalService) nextEventSequence(ctx context.Context, accountID, runID string) (uint64, error) {
	type eventLister interface {
		ListEventsAfter(context.Context, string, string, uint64, int) ([]*domain.Event, error)
	}
	repo, ok := s.Repo.(eventLister)
	if !ok {
		return 1, nil
	}
	events, err := repo.ListEventsAfter(ctx, accountID, runID, 0, 200)
	if err != nil {
		return 0, err
	}
	var sequence uint64
	for _, event := range events {
		if event.Sequence > sequence {
			sequence = event.Sequence
		}
	}
	return sequence + 1, nil
}

func (s *ApprovalService) nextID() string {
	if s.IDs != nil {
		return s.IDs()
	}
	return uuid.NewString()
}

func (s *ApprovalService) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}
