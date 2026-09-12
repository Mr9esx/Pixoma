package botapp_test

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	domain "github.com/Mr9esx/Pixoma/internal/cases/domain"
	"github.com/Mr9esx/Pixoma/internal/cases/infrastructure/validation"
	"github.com/Mr9esx/Pixoma/internal/packaging/botapp"
	"github.com/Mr9esx/Pixoma/internal/platform/blob/localfs"
	convdomain "github.com/Mr9esx/Pixoma/internal/sessions/domain"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	runtimedomain "github.com/Mr9esx/Pixoma/internal/tasks/domain"
	identitydomain "github.com/Mr9esx/Pixoma/internal/users/domain"
)

type memUsers struct {
	byID map[string]*identitydomain.User
}

func (r *memUsers) UpsertByChannelExternal(context.Context, identitydomain.UpsertFrom) (*identitydomain.User, error) {
	return nil, errors.New("unused")
}
func (r *memUsers) SetAccess(context.Context, string, identitydomain.UserAccess) (*identitydomain.User, error) {
	return nil, errors.New("unused")
}
func (r *memUsers) List(context.Context, identitydomain.ListQuery) ([]*identitydomain.User, error) {
	return nil, nil
}
func (r *memUsers) Delete(context.Context, string) error {
	return errors.New("unused")
}
func (r *memUsers) GetByID(_ context.Context, id string) (*identitydomain.User, error) {
	u, ok := r.byID[id]
	if !ok {
		return nil, identitydomain.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func mcpRunFacade(t *testing.T, access identitydomain.UserAccess) (*botapp.Facade, *capturePub, *runtimedomain.MemoryTaskRepository, *memCases) {
	t.Helper()
	ctx := context.Background()
	cases := &memCases{}
	_ = cases.Create(ctx, &domain.Case{Document: sampleDoc(), Enabled: true})
	pub := &capturePub{}
	tasks := runtimedomain.NewMemoryTaskRepository()
	store, err := localfs.New(filepath.Join(t.TempDir(), "blob"))
	if err != nil {
		t.Fatal(err)
	}
	user := &identitydomain.User{
		ID:             "user-mcp-1",
		ChannelID:      "ch-mcp",
		ExternalUserID: "ext-1",
		Access:         access,
	}
	sessRepo := convdomain.NewMemoryRepository()
	facade := &botapp.Facade{
		Cases:        cases,
		Validator:    validation.New(),
		Users:        &memUsers{byID: map[string]*identitydomain.User{"user-mcp-1": user}},
		SessionStore: sessRepo,
		Tasks:        tasks,
		Blob:         store,
		Publisher:    pub,
		NewTaskID:    func() sharedkernel.TaskID { return "task-mcp-1" },
		Now:          func() time.Time { return time.Unix(20, 0).UTC() },
	}
	return facade, pub, tasks, cases
}

func TestRunCase_CreatesPendingAndPublishes(t *testing.T) {
	ctx := context.Background()
	facade, pub, tasks, _ := mcpRunFacade(t, identitydomain.UserAccessAlwaysAllowed)
	prompt := "a cat"
	chat := sharedkernel.ChatID(sharedkernel.FormatChatID(sharedkernel.ChannelAddr{ChannelID: "ch-mcp", ExternalChatID: "ext-1"}))

	res, err := facade.RunCase(ctx, botapp.RunCaseCmd{
		CaseID: 1,
		Inputs: []domain.InputValue{{Key: "prompt", Text: &prompt}},
		Actor:  chat,
		UserID: "user-mcp-1",
	})
	if err != nil {
		t.Fatalf("RunCase: %v", err)
	}
	if res.TaskID != "task-mcp-1" || res.Status != sharedkernel.TaskPending {
		t.Fatalf("result=%+v", res)
	}
	got, err := tasks.Get(ctx, "task-mcp-1")
	if err != nil || got.Status != sharedkernel.TaskPending {
		t.Fatalf("task=%v err=%v", got, err)
	}
	if got.ChatID != chat {
		t.Fatalf("ChatID=%q want %q", got.ChatID, chat)
	}
	if got.SessionID == "" {
		t.Fatal("expected session_id")
	}
	sess, err := facade.SessionStore.GetByID(ctx, got.SessionID)
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	if sess.UserID != "user-mcp-1" || sess.ChatID != chat || sess.Status != convdomain.StatusSubmitted {
		t.Fatalf("session=%+v", sess)
	}
	if len(pub.msgs) != 1 || pub.msgs[0].Topic != sharedkernel.TopicTaskCreated {
		t.Fatalf("msgs=%+v", pub.msgs)
	}
	var ev sharedkernel.TaskCreated
	if err := json.Unmarshal(pub.msgs[0].Payload, &ev); err != nil {
		t.Fatal(err)
	}
	if ev.TaskID != "task-mcp-1" || ev.ChatID != chat {
		t.Fatalf("event=%+v", ev)
	}
}

func TestRunCase_RejectsMissingRequired(t *testing.T) {
	ctx := context.Background()
	facade, pub, tasks, _ := mcpRunFacade(t, identitydomain.UserAccessAlwaysAllowed)
	chat := sharedkernel.ChatID(sharedkernel.FormatChatID(sharedkernel.ChannelAddr{ChannelID: "ch-mcp", ExternalChatID: "ext-1"}))

	_, err := facade.RunCase(ctx, botapp.RunCaseCmd{
		CaseID: 1,
		Inputs: nil,
		Actor:  chat,
		UserID: "user-mcp-1",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if _, getErr := tasks.Get(ctx, "task-mcp-1"); getErr == nil {
		t.Fatal("must not create task")
	}
	if len(pub.msgs) != 0 {
		t.Fatalf("published=%+v", pub.msgs)
	}
}

func TestRunCase_RejectsDeniedUser(t *testing.T) {
	ctx := context.Background()
	facade, pub, tasks, _ := mcpRunFacade(t, identitydomain.UserAccessDenied)
	prompt := "a cat"
	chat := sharedkernel.ChatID(sharedkernel.FormatChatID(sharedkernel.ChannelAddr{ChannelID: "ch-mcp", ExternalChatID: "ext-1"}))

	_, err := facade.RunCase(ctx, botapp.RunCaseCmd{
		CaseID: 1,
		Inputs: []domain.InputValue{{Key: "prompt", Text: &prompt}},
		Actor:  chat,
		UserID: "user-mcp-1",
	})
	if !errors.Is(err, botapp.ErrAccessDenied) {
		t.Fatalf("want ErrAccessDenied, got %v", err)
	}
	if _, getErr := tasks.Get(ctx, "task-mcp-1"); getErr == nil {
		t.Fatal("must not create task")
	}
	if len(pub.msgs) != 0 {
		t.Fatalf("published=%+v", pub.msgs)
	}
}

func TestRunCase_WorksWithoutTelegram(t *testing.T) {
	ctx := context.Background()
	facade, _, tasks, _ := mcpRunFacade(t, identitydomain.UserAccessAlwaysAllowed)
	prompt := "a cat"
	chat := sharedkernel.ChatID(sharedkernel.FormatChatID(sharedkernel.ChannelAddr{ChannelID: "ch-mcp", ExternalChatID: "ext-1"}))

	res, err := facade.RunCase(ctx, botapp.RunCaseCmd{
		CaseID: 1,
		Inputs: []domain.InputValue{{Key: "prompt", Text: &prompt}},
		Actor:  chat,
		UserID: "user-mcp-1",
	})
	if err != nil {
		t.Fatalf("RunCase: %v", err)
	}
	got, err := tasks.Get(ctx, res.TaskID)
	if err != nil || got.Status != sharedkernel.TaskPending {
		t.Fatalf("task=%v err=%v", got, err)
	}
}
