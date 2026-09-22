package domain

import (
	"fmt"
	"strings"
	"time"
)

type WorkflowExecutionStatus string

const (
	WorkflowExecutionSubmitted WorkflowExecutionStatus = "submitted"
	WorkflowExecutionSucceeded WorkflowExecutionStatus = "succeeded"
	WorkflowExecutionFailed    WorkflowExecutionStatus = "failed"
	WorkflowExecutionCancelled WorkflowExecutionStatus = "cancelled"
)

func (s WorkflowExecutionStatus) Terminal() bool {
	return s == WorkflowExecutionSucceeded || s == WorkflowExecutionFailed || s == WorkflowExecutionCancelled
}

// WorkflowExecution is the durable, account-scoped bridge between one Studio
// Agent tool invocation and one asynchronous Pixoma task.
type WorkflowExecution struct {
	ID              string
	AccountID       string
	SessionID       string
	RunID           string
	ToolCallID      string
	TaskID          string
	WorkflowID      string
	OperationNodeID string
	Status          WorkflowExecutionStatus
	ErrorMessage    string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	CompletedAt     time.Time
}

func NewWorkflowExecution(id, accountID, sessionID, runID, toolCallID, taskID, workflowID, operationNodeID string, now time.Time) (*WorkflowExecution, error) {
	if anyBlank(id, accountID, sessionID, runID, toolCallID, taskID, workflowID, operationNodeID) {
		return nil, fmt.Errorf("%w: invalid workflow execution", ErrInvalid)
	}
	now = now.UTC()
	return &WorkflowExecution{
		ID: id, AccountID: strings.TrimSpace(accountID), SessionID: strings.TrimSpace(sessionID),
		RunID: strings.TrimSpace(runID), ToolCallID: strings.TrimSpace(toolCallID), TaskID: strings.TrimSpace(taskID),
		WorkflowID: strings.TrimSpace(workflowID), OperationNodeID: strings.TrimSpace(operationNodeID),
		Status: WorkflowExecutionSubmitted, CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (e *WorkflowExecution) Complete(status WorkflowExecutionStatus, errorMessage string, now time.Time) error {
	if e == nil || !status.Terminal() || e.Status != WorkflowExecutionSubmitted {
		return ErrInvalidTransition
	}
	now = now.UTC()
	e.Status = status
	e.ErrorMessage = strings.TrimSpace(errorMessage)
	e.UpdatedAt = now
	e.CompletedAt = now
	return nil
}
