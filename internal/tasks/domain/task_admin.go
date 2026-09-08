package domain

import (
	"context"

	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

type TaskAdminUser struct {
	ID             string
	ChannelID      string
	ExternalUserID string
	Username       string
	FirstName      string
	LastName       string
}

type TaskAdminContext struct {
	Task        *Task
	ChannelID   string
	ChannelName string
	UserID      string
	User        *TaskAdminUser
}

type TaskAdminReader interface {
	List(ctx context.Context, q AdminListQuery) ([]*TaskAdminContext, error)
	Get(ctx context.Context, id sharedkernel.TaskID) (*TaskAdminContext, error)
}

func NewTaskAdminContext(task *Task) *TaskAdminContext {
	return &TaskAdminContext{Task: task}
}
