package domain

import (
	"errors"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

var (
	ErrInvalidTransition = errors.New("invalid task transition")
	ErrCancelNotAllowed  = errors.New("cancel not allowed in current status")
)

type OutputRef struct {
	Key  string
	Blob sharedkernel.BlobRef
}

type Task struct {
	ID           sharedkernel.TaskID
	ChatID       sharedkernel.ChatID
	CaseID       sharedkernel.CaseID
	Status       sharedkernel.TaskStatus
	InstanceID   sharedkernel.InstanceID
	PromptID     string
	InputPrefix  string
	Outputs      []OutputRef
	ErrorCode    string
	ErrorMessage string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewPending(id sharedkernel.TaskID, chat sharedkernel.ChatID, caseID sharedkernel.CaseID, inputPrefix string, now time.Time) *Task {
	return &Task{
		ID:          id,
		ChatID:      chat,
		CaseID:      caseID,
		Status:      sharedkernel.TaskPending,
		InputPrefix: inputPrefix,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func (t *Task) MarkQueued(instance sharedkernel.InstanceID, now time.Time) error {
	if t.Status != sharedkernel.TaskPending && t.Status != sharedkernel.TaskQueued {
		return ErrInvalidTransition
	}
	t.Status = sharedkernel.TaskQueued
	t.InstanceID = instance
	t.UpdatedAt = now
	return nil
}

func (t *Task) MarkRunning(promptID string, now time.Time) error {
	if t.Status != sharedkernel.TaskQueued && t.Status != sharedkernel.TaskRunning {
		return ErrInvalidTransition
	}
	t.Status = sharedkernel.TaskRunning
	if promptID != "" {
		t.PromptID = promptID
	}
	t.UpdatedAt = now
	return nil
}

func (t *Task) MarkSucceeded(outputs []OutputRef, now time.Time) error {
	if t.Status == sharedkernel.TaskSucceeded {
		return nil // idempotent
	}
	if t.Status != sharedkernel.TaskRunning && t.Status != sharedkernel.TaskQueued {
		return ErrInvalidTransition
	}
	t.Status = sharedkernel.TaskSucceeded
	t.Outputs = append([]OutputRef(nil), outputs...)
	t.UpdatedAt = now
	return nil
}

func (t *Task) MarkFailed(code, msg string, now time.Time) error {
	if t.Status == sharedkernel.TaskFailed {
		return nil
	}
	if t.Status == sharedkernel.TaskSucceeded || t.Status == sharedkernel.TaskCancelled {
		return ErrInvalidTransition
	}
	t.Status = sharedkernel.TaskFailed
	t.ErrorCode = code
	t.ErrorMessage = msg
	t.UpdatedAt = now
	return nil
}

// MarkCancelled implements mild cancel: only pending or queued.
func (t *Task) MarkCancelled(now time.Time) error {
	if t.Status == sharedkernel.TaskCancelled {
		return nil
	}
	if t.Status != sharedkernel.TaskPending && t.Status != sharedkernel.TaskQueued {
		return ErrCancelNotAllowed
	}
	t.Status = sharedkernel.TaskCancelled
	t.UpdatedAt = now
	return nil
}
