package botapp

import (
	"context"

	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type StartCaseCmd struct {
	ChatID    sharedkernel.ChatID
	CaseID    sharedkernel.CaseID
	InputKeys []string
}

type SessionView struct {
	SessionID sharedkernel.SessionID
	CaseID    sharedkernel.CaseID
	Status    convdomain.Status
	Index     int
	Keys      []string
}

func (f *Facade) StartCase(ctx context.Context, cmd StartCaseCmd) (*SessionView, error) {
	c, err := f.Cases.Get(ctx, cmd.CaseID)
	if err != nil {
		return nil, err
	}
	if !c.Enabled {
		return nil, catalogdomain.ErrDisabled
	}
	keys := cmd.InputKeys
	if len(keys) == 0 {
		for _, in := range c.Document.Inputs {
			keys = append(keys, in.Key)
		}
	}
	s, err := f.Sessions.StartCase(ctx, cmd.ChatID, cmd.CaseID, keys)
	if err != nil {
		return nil, err
	}
	return toView(s), nil
}

func (f *Facade) SubmitInput(ctx context.Context, chatID sharedkernel.ChatID, v convdomain.DraftValue) (*SessionView, error) {
	s, err := f.Sessions.SubmitInput(ctx, chatID, v)
	if err != nil {
		return nil, err
	}
	return toView(s), nil
}

func (f *Facade) SkipInput(ctx context.Context, chatID sharedkernel.ChatID) (*SessionView, error) {
	s, err := f.Sessions.SkipInput(ctx, chatID)
	if err != nil {
		return nil, err
	}
	return toView(s), nil
}

func (f *Facade) ExitSession(ctx context.Context, chatID sharedkernel.ChatID) error {
	return f.Sessions.Exit(ctx, chatID)
}

func (f *Facade) GetSession(ctx context.Context, chatID sharedkernel.ChatID) (*SessionView, error) {
	s, err := f.Sessions.Get(ctx, chatID)
	if err != nil {
		return nil, err
	}
	return toView(s), nil
}

func (f *Facade) ListCases(ctx context.Context, q catalogdomain.ListQuery) ([]*catalogdomain.Case, error) {
	return f.Cases.List(ctx, q)
}

func (f *Facade) ListMyTasks(ctx context.Context, chatID sharedkernel.ChatID, limit int) ([]*runtimedomain.Task, error) {
	return f.Tasks.ListByChat(ctx, chatID, limit)
}

func toView(s *convdomain.Session) *SessionView {
	return &SessionView{
		SessionID: s.ID,
		CaseID:    s.CaseID,
		Status:    s.Status,
		Index:     s.CurrentInputIndex,
		Keys:      append([]string(nil), s.InputKeys...),
	}
}
