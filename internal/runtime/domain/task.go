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
	SessionID    sharedkernel.SessionID
	ChatID       sharedkernel.ChatID // optional cache; not persisted as required column
	CaseID       sharedkernel.CaseID
	Status       sharedkernel.TaskStatus
	InstanceID   sharedkernel.InstanceID
	PromptID     string
	InputPrefix  string
	JobRef       sharedkernel.BlobRef
	LeaseUntil   time.Time
	Outputs      []OutputRef
	ErrorCode    string
	ErrorMessage string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewPending(id sharedkernel.TaskID, sessionID sharedkernel.SessionID, caseID sharedkernel.CaseID, inputPrefix string, now time.Time) *Task {
	return &Task{
		ID:          id,
		SessionID:   sessionID,
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

// PrepareForClaim marks a pending task queued for a specific instance with a job blob ref.
func (t *Task) PrepareForClaim(instance sharedkernel.InstanceID, jobRef sharedkernel.BlobRef, now time.Time) error {
	if t.Status != sharedkernel.TaskPending && t.Status != sharedkernel.TaskQueued {
		return ErrInvalidTransition
	}
	if instance == "" || jobRef.Key == "" {
		return ErrInvalidTransition
	}
	t.Status = sharedkernel.TaskQueued
	t.InstanceID = instance
	t.JobRef = jobRef
	t.LeaseUntil = time.Time{}
	t.UpdatedAt = now
	return nil
}

// ClaimWithLease moves queued → running for the assigned instance and sets lease expiry.
func (t *Task) ClaimWithLease(instance sharedkernel.InstanceID, lease time.Duration, now time.Time) error {
	if t.Status != sharedkernel.TaskQueued {
		return ErrInvalidTransition
	}
	if instance == "" || t.InstanceID != instance {
		return ErrInvalidTransition
	}
	if t.JobRef.Key == "" {
		return ErrInvalidTransition
	}
	if lease <= 0 {
		return ErrInvalidTransition
	}
	t.Status = sharedkernel.TaskRunning
	t.LeaseUntil = now.Add(lease)
	t.UpdatedAt = now
	return nil
}

// RequeueIfLeaseExpired returns queued to the same instance when a running lease is past due.
func (t *Task) RequeueIfLeaseExpired(now time.Time) (bool, error) {
	if t.Status != sharedkernel.TaskRunning {
		return false, nil
	}
	if t.LeaseUntil.IsZero() || !now.After(t.LeaseUntil) {
		return false, nil
	}
	t.Status = sharedkernel.TaskQueued
	t.LeaseUntil = time.Time{}
	t.UpdatedAt = now
	return true, nil
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
