package botapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	catalogdomain "github.com/Mr9esx/Pixoma/internal/cases/domain"
	convdomain "github.com/Mr9esx/Pixoma/internal/sessions/domain"
	"github.com/Mr9esx/Pixoma/internal/platform/queue"
	runtimedomain "github.com/Mr9esx/Pixoma/internal/tasks/domain"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	identitydomain "github.com/Mr9esx/Pixoma/internal/users/domain"
)

var ErrAccessDenied = errors.New("user access denied")

type RunCaseCmd struct {
	CaseID sharedkernel.CaseID
	Inputs []catalogdomain.InputValue
	Actor  sharedkernel.ChatID
	UserID string
}

func (f *Facade) RunCase(ctx context.Context, cmd RunCaseCmd) (*ConfirmRunResult, error) {
	if f.SessionStore == nil || f.NewTaskID == nil || f.Users == nil {
		return nil, fmt.Errorf("botapp: incomplete facade wiring")
	}
	if cmd.UserID == "" {
		return nil, convdomain.ErrEmptyUserID
	}
	user, err := f.Users.GetByID(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.Access != identitydomain.UserAccessAlwaysAllowed {
		return nil, ErrAccessDenied
	}

	c, err := f.Cases.Get(ctx, cmd.CaseID)
	if err != nil {
		return nil, err
	}
	if !c.Enabled {
		return nil, catalogdomain.ErrDisabled
	}
	if err := f.Validator.ValidateInputs(c.Document, cmd.Inputs); err != nil {
		return nil, err
	}

	now := f.now()
	taskID := f.NewTaskID()
	inputPrefix := fmt.Sprintf("inputs/%s", taskID)
	if f.Blob != nil {
		if err := stageInputs(ctx, f.Blob, inputPrefix, cmd.Inputs); err != nil {
			return nil, fmt.Errorf("stage inputs: %w", err)
		}
	}

	sessID := sharedkernel.SessionID("mcp-" + string(taskID))
	sess := &convdomain.Session{
		ID:        sessID,
		UserID:    cmd.UserID,
		ChatID:    cmd.Actor,
		CaseID:    cmd.CaseID,
		Status:    convdomain.StatusSubmitted,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if addr, err := sharedkernel.ParseChatID(string(cmd.Actor)); err == nil {
		sess.ChannelID = addr.ChannelID
	}
	if err := f.SessionStore.Save(ctx, sess); err != nil {
		return nil, err
	}

	task := runtimedomain.NewPending(taskID, sessID, cmd.CaseID, inputPrefix, now)
	task.ChatID = cmd.Actor
	if err := f.Tasks.Create(ctx, task); err != nil {
		return nil, err
	}

	payload, err := json.Marshal(sharedkernel.TaskCreated{
		TaskID:    taskID,
		ChatID:    cmd.Actor,
		CaseID:    cmd.CaseID,
		CreatedAt: now,
	})
	if err != nil {
		return nil, err
	}
	if err := f.Publisher.Publish(ctx, queue.Message{
		Topic:   sharedkernel.TopicTaskCreated,
		Key:     string(taskID),
		Payload: payload,
	}); err != nil {
		return nil, err
	}
	return &ConfirmRunResult{TaskID: taskID, Status: sharedkernel.TaskPending}, nil
}
