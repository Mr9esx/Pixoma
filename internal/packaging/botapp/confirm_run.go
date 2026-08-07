package botapp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type ConfirmRunCmd struct {
	ChatID sharedkernel.ChatID
}

type ConfirmRunResult struct {
	TaskID sharedkernel.TaskID
	Status sharedkernel.TaskStatus
}

type Facade struct {
	Cases        catalogdomain.Repository
	Validator    catalogdomain.Validator
	Sessions     *convdomain.Service
	SessionStore convdomain.Repository
	Tasks        runtimedomain.TaskRepository
	Publisher    queue.Publisher
	NewTaskID    func() sharedkernel.TaskID
	Now          func() time.Time
}

func (f *Facade) ConfirmRun(ctx context.Context, cmd ConfirmRunCmd) (*ConfirmRunResult, error) {
	if f.SessionStore == nil || f.NewTaskID == nil {
		return nil, fmt.Errorf("botapp: incomplete facade wiring")
	}
	now := f.now()

	sess, err := f.Sessions.Get(ctx, cmd.ChatID)
	if err != nil {
		return nil, err
	}
	if sess.Status != convdomain.StatusConfirming {
		return nil, convdomain.ErrInvalidState
	}

	c, err := f.Cases.Get(ctx, sess.CaseID)
	if err != nil {
		return nil, err
	}
	if !c.Enabled {
		return nil, catalogdomain.ErrDisabled
	}

	values := draftToInputValues(sess)
	if err := f.Validator.ValidateInputs(c.Document, values); err != nil {
		return nil, err
	}

	taskID := f.NewTaskID()
	task := runtimedomain.NewPending(taskID, cmd.ChatID, sess.CaseID, fmt.Sprintf("inputs/%s", taskID), now)
	if err := f.Tasks.Create(ctx, task); err != nil {
		return nil, err
	}

	if err := sess.MarkSubmitted(now); err != nil {
		return nil, err
	}
	if err := f.SessionStore.Save(ctx, sess); err != nil {
		return nil, err
	}

	payload, err := json.Marshal(sharedkernel.TaskCreated{
		TaskID:    taskID,
		ChatID:    cmd.ChatID,
		CaseID:    sess.CaseID,
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

func (f *Facade) now() time.Time {
	if f.Now != nil {
		return f.Now()
	}
	return time.Now().UTC()
}

func draftToInputValues(sess *convdomain.Session) []catalogdomain.InputValue {
	out := make([]catalogdomain.InputValue, 0, len(sess.Draft))
	for _, key := range sess.InputKeys {
		d, ok := sess.Draft[key]
		if !ok || d.Skipped {
			continue
		}
		out = append(out, catalogdomain.InputValue{
			Key:    key,
			Text:   d.Text,
			Number: d.Number,
			Bool:   d.Bool,
			Blob:   d.Blob,
		})
	}
	return out
}
