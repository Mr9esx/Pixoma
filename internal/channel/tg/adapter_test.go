package tg_test

import (
	"context"
	"strings"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/validation"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/tg"
	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/packaging/botapp"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type memOut struct {
	texts []string
}

func (m *memOut) SendText(_ context.Context, _ int64, text string) error {
	m.texts = append(m.texts, text)
	return nil
}
func (m *memOut) SendPhoto(_ context.Context, _ int64, _ sharedkernel.BlobRef, caption string) error {
	m.texts = append(m.texts, "photo:"+caption)
	return nil
}

type memCases struct{ c *domain.Case }

func (m *memCases) Save(ctx context.Context, c *domain.Case) error { return m.Create(ctx, c) }
func (m *memCases) Create(_ context.Context, c *domain.Case) error  { m.c = c; return nil }
func (m *memCases) Get(_ context.Context, id sharedkernel.CaseID) (*domain.Case, error) {
	if m.c == nil || m.c.Document.ID != id {
		return nil, domain.ErrNotFound
	}
	cp := *m.c
	return &cp, nil
}
func (m *memCases) List(context.Context, domain.ListQuery) ([]*domain.Case, error) {
	if m.c == nil {
		return nil, nil
	}
	cp := *m.c
	return []*domain.Case{&cp}, nil
}
func (m *memCases) Disable(context.Context, sharedkernel.CaseID) error { return nil }

func TestHandleTextLockMessage(t *testing.T) {
	ctx := context.Background()
	cases := &memCases{}
	_ = cases.Create(ctx, &domain.Case{
		Enabled: true,
		Document: domain.CaseDocument{
			ID: "c1", Name: "Demo",
			Inputs:    []domain.InputField{{Key: "prompt", Type: "string", Required: true}},
			Bindings:  domain.ComfyBindings{WorkflowJSON: map[string]any{"1": map[string]any{}}},
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{
				"prompt": map[string]any{"type": "string"},
			}},
		},
	})
	sessRepo := convdomain.NewMemoryRepository()
	sessSvc := convdomain.NewService(sessRepo, func() sharedkernel.SessionID { return "s" }, nil)
	facade := &botapp.Facade{
		Cases: cases, Validator: validation.New(), Sessions: sessSvc, SessionStore: sessRepo,
		Tasks:     runtimedomain.NewMemoryTaskRepository(),
		NewTaskID: func() sharedkernel.TaskID { return "t" },
	}
	out := &memOut{}
	ad := tg.New(facade, out)
	if err := ad.HandleText(ctx, 1, "/start_case c1"); err != nil {
		t.Fatal(err)
	}
	if err := ad.HandleText(ctx, 1, "/start_case c1"); err != nil {
		t.Fatal(err)
	}
	last := out.texts[len(out.texts)-1]
	if !strings.Contains(last, "进行中") {
		t.Fatalf("want lock message, got %q", last)
	}
}

func TestNotifyDedupe(t *testing.T) {
	ctx := context.Background()
	out := &memOut{}
	ad := tg.New(&botapp.Facade{}, out)
	n := sharedkernel.UserNotify{ChatID: 1, TaskID: "t1", Kind: "task_failed", ErrorMsg: "x"}
	_ = ad.HandleUserNotify(ctx, n)
	_ = ad.HandleUserNotify(ctx, n)
	if len(out.texts) != 1 {
		t.Fatalf("dedupe failed: %v", out.texts)
	}
}
