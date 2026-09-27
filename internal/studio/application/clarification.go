package application

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type ClarificationRepository interface {
	GetClarification(ctx context.Context, accountID, clarificationID string) (*domain.Clarification, error)
	UpdateClarification(ctx context.Context, clarification *domain.Clarification) error
	GetRun(ctx context.Context, accountID, runID string) (*domain.Run, error)
	UpdateRun(ctx context.Context, run *domain.Run) error
	AppendRunEvent(ctx context.Context, event *domain.Event) (*domain.Event, error)
}

type ClarificationService struct {
	Repo        ClarificationRepository
	Queue       RunQueue
	Checkpoints interface {
		Get(context.Context, string) ([]byte, bool, error)
	}
	IDs func() string
	Now func() time.Time
}

type ResolveClarificationInput struct {
	AccountID       string
	ClarificationID string
	Selected        string
	Custom          string
	WorkflowInputs  map[string]any
	SkipWorkflow    bool
}

func (s *ClarificationService) Resolve(ctx context.Context, input ResolveClarificationInput) error {
	if s == nil || s.Repo == nil || s.Queue == nil || s.Checkpoints == nil {
		return fmt.Errorf("studio: clarification service is not configured")
	}
	clarification, err := s.Repo.GetClarification(ctx, input.AccountID, input.ClarificationID)
	if err != nil {
		return err
	}
	run, err := s.Repo.GetRun(ctx, input.AccountID, clarification.RunID)
	if err != nil {
		return err
	}
	if run.Status != domain.RunWaitingClarification {
		return domain.ErrInvalidTransition
	}
	now := s.now()
	_, exists, err := s.Checkpoints.Get(ctx, run.ID)
	if err != nil {
		return err
	}
	if !exists {
		if err := run.Fail("checkpoint_missing", "澄清检查点已丢失", now); err != nil {
			return err
		}
		if err := s.Repo.UpdateRun(ctx, run); err != nil {
			return err
		}
		return fmt.Errorf("%w: clarification checkpoint missing", domain.ErrInvalidTransition)
	}
	if clarification.Workflow != nil {
		if !input.SkipWorkflow {
			if err := validateWorkflowResponse(clarification.Workflow.InputSchema, input.WorkflowInputs); err != nil {
				return err
			}
		}
		if err := clarification.ResolveWorkflow(input.AccountID, input.WorkflowInputs, input.SkipWorkflow, now); err != nil {
			return err
		}
	} else if err := clarification.Resolve(input.AccountID, input.Selected, input.Custom, now); err != nil {
		return err
	}
	if err := s.Repo.UpdateClarification(ctx, clarification); err != nil {
		return err
	}
	if err := run.Resume(now); err != nil {
		return err
	}
	if err := s.Repo.UpdateRun(ctx, run); err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]any{
		"clarification_id": clarification.ID, "selected": clarification.Selected, "answer": clarification.Answer,
	})
	if err != nil {
		return err
	}
	eventType := EventClarificationAnswered
	if clarification.Status == domain.ClarificationSkipped {
		eventType = EventClarificationSkipped
	}
	if _, err := s.Repo.AppendRunEvent(ctx, &domain.Event{
		ID: s.nextID(), RunID: run.ID, SessionID: run.SessionID, AccountID: run.AccountID,
		Type: eventType, Payload: payload, CreatedAt: now,
	}); err != nil {
		return err
	}
	return s.Queue.Enqueue(RunRef{AccountID: input.AccountID, RunID: run.ID})
}

func validateWorkflowResponse(raw json.RawMessage, inputs map[string]any) error {
	if inputs == nil {
		return fmt.Errorf("%w: workflow inputs are required", domain.ErrInvalid)
	}
	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("%w: invalid workflow schema: %v", domain.ErrInvalid, err)
	}
	compiler := jsonschema.NewCompiler()
	const url = "mem://studio-workflow-inputs.json"
	if err := compiler.AddResource(url, document); err != nil {
		return err
	}
	schema, err := compiler.Compile(url)
	if err != nil {
		return err
	}
	if err := schema.Validate(inputs); err != nil {
		return fmt.Errorf("%w: invalid workflow inputs: %v", domain.ErrInvalid, err)
	}
	return nil
}

func (s *ClarificationService) nextID() string {
	if s.IDs != nil {
		return s.IDs()
	}
	return uuid.NewString()
}

func (s *ClarificationService) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}
