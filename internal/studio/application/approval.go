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
	AppendRunEvent(ctx context.Context, event *domain.Event) (*domain.Event, error)
}

type ApprovalService struct {
	Repo        ApprovalRepository
	Queue       RunQueue
	Checkpoints interface {
		Get(context.Context, string) ([]byte, bool, error)
	}
	IDs func() string
	Now func() time.Time
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
	run, err := s.Repo.GetRun(ctx, input.AccountID, approval.RunID)
	if err != nil {
		return err
	}
	if run.Status != domain.RunWaitingApproval {
		return domain.ErrInvalidTransition
	}
	now := s.now()
	if input.Approved && s.Checkpoints != nil {
		_, exists, checkpointErr := s.Checkpoints.Get(ctx, run.ID)
		if checkpointErr != nil {
			return checkpointErr
		}
		if !exists {
			if err := run.Fail("checkpoint_missing", "批准检查点已丢失，请手动重试", now); err != nil {
				return err
			}
			if err := s.Repo.UpdateRun(ctx, run); err != nil {
				return err
			}
			return fmt.Errorf("%w: approval checkpoint missing; run interrupted", domain.ErrInvalidTransition)
		}
	}
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
	status := "rejected"
	if input.Approved {
		status = "approved"
	}
	payload := []byte(fmt.Sprintf(`{"approval_id":%q,"status":%q}`, approval.ID, status))
	if _, err := s.Repo.AppendRunEvent(ctx, &domain.Event{
		ID: s.nextID(), RunID: run.ID, SessionID: run.SessionID, AccountID: run.AccountID,
		Type: EventApprovalResolved, Payload: payload, CreatedAt: now,
	}); err != nil {
		return err
	}
	if !input.Approved {
		return nil
	}
	return s.Queue.Enqueue(RunRef{AccountID: input.AccountID, RunID: run.ID})
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
