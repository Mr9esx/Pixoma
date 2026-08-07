package tg_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/validation"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/tg"
	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/packaging/botapp"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type memOut struct {
	texts   []string
	menus   []string
	inlines []string
	photos  []string
}

func (m *memOut) SendText(_ context.Context, _ int64, text string) error {
	m.texts = append(m.texts, text)
	return nil
}
func (m *memOut) SendMenu(_ context.Context, _ int64, text string) error {
	m.menus = append(m.menus, text)
	return nil
}
func (m *memOut) SendInline(_ context.Context, _ int64, text string, _ [][]tg.InlineButton) error {
	m.inlines = append(m.inlines, text)
	return nil
}
func (m *memOut) SendPhoto(_ context.Context, _ int64, ref sharedkernel.BlobRef, caption string) error {
	m.photos = append(m.photos, caption+"|"+ref.Key)
	return nil
}
func (m *memOut) AnswerCallback(context.Context, string, string) error { return nil }

type memCases struct {
	items map[sharedkernel.CaseID]*domain.Case
}

func (m *memCases) ensure() {
	if m.items == nil {
		m.items = map[sharedkernel.CaseID]*domain.Case{}
	}
}
func (m *memCases) Save(ctx context.Context, c *domain.Case) error { return m.Create(ctx, c) }
func (m *memCases) Create(_ context.Context, c *domain.Case) error {
	m.ensure()
	cp := *c
	m.items[c.Document.ID] = &cp
	return nil
}
func (m *memCases) Get(_ context.Context, id sharedkernel.CaseID) (*domain.Case, error) {
	m.ensure()
	c, ok := m.items[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *c
	return &cp, nil
}
func (m *memCases) List(_ context.Context, q domain.ListQuery) ([]*domain.Case, error) {
	m.ensure()
	var out []*domain.Case
	for _, c := range m.items {
		if q.Tag != "" {
			ok := false
			for _, t := range c.Document.Tags {
				if t == q.Tag {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		cp := *c
		out = append(out, &cp)
	}
	return out, nil
}
func (m *memCases) Disable(context.Context, sharedkernel.CaseID) error { return nil }

type discardPub struct{}

func (discardPub) Publish(context.Context, queue.Message) error { return nil }

func sampleCase(id, name string) *domain.Case {
	return &domain.Case{
		Enabled: true,
		Document: domain.CaseDocument{
			ID: sharedkernel.CaseID(id), Name: name, Description: "desc", Preview: "preview", Price: 10,
			Tags: []string{"image"},
			Inputs: []domain.InputField{
				{Key: "prompt", Type: "string", Required: true, Description: "画面描述"},
			},
			Bindings: domain.ComfyBindings{WorkflowJSON: map[string]any{"1": map[string]any{}}},
			InputSchema: map[string]any{
				"type": "object", "required": []any{"prompt"},
				"properties": map[string]any{"prompt": map[string]any{"type": "string", "minLength": 1}},
			},
		},
	}
}

func newFacade(cases *memCases) *botapp.Facade {
	sessRepo := convdomain.NewMemoryRepository()
	sessSvc := convdomain.NewService(sessRepo, func() sharedkernel.SessionID { return "s" }, func() time.Time {
		return time.Unix(1, 0).UTC()
	})
	return &botapp.Facade{
		Cases: cases, Validator: validation.New(), Sessions: sessSvc, SessionStore: sessRepo,
		Tasks: runtimedomain.NewMemoryTaskRepository(), Publisher: discardPub{},
		NewTaskID: func() sharedkernel.TaskID { return "t1" },
		Now:       func() time.Time { return time.Unix(2, 0).UTC() },
	}
}

func TestStartShowsMenu(t *testing.T) {
	out := &memOut{}
	ad := tg.New(newFacade(&memCases{}), out)
	if err := ad.HandleText(context.Background(), 1, "/start"); err != nil {
		t.Fatal(err)
	}
	if len(out.menus) == 0 || !strings.Contains(out.menus[0], "欢迎") {
		t.Fatalf("menus=%v", out.menus)
	}
}

func TestImageListAndPreviewAndFlow(t *testing.T) {
	ctx := context.Background()
	cases := &memCases{}
	_ = cases.Create(ctx, sampleCase("img-anime", "二次元文生图"))
	_ = cases.Create(ctx, sampleCase("img-logo", "Logo 草图"))
	out := &memOut{}
	ad := tg.New(newFacade(cases), out)

	if err := ad.HandleText(ctx, 1, tg.BtnImage); err != nil {
		t.Fatal(err)
	}
	if len(out.inlines) == 0 || !strings.Contains(out.inlines[0], "图片 Case") {
		t.Fatalf("list=%v", out.inlines)
	}

	if err := ad.HandleCallback(ctx, 1, "cb1", tg.CBCasePreview+"img-anime"); err != nil {
		t.Fatal(err)
	}
	last := out.inlines[len(out.inlines)-1]
	if !strings.Contains(last, "二次元") || !strings.Contains(last, "preview") {
		t.Fatalf("preview=%q", last)
	}

	if err := ad.HandleCallback(ctx, 1, "cb2", tg.CBCaseStart+"img-anime"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.inlines[len(out.inlines)-1], "prompt") {
		t.Fatalf("start prompt=%q", out.inlines[len(out.inlines)-1])
	}

	if err := ad.HandleText(ctx, 1, "a cute cat"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.inlines[len(out.inlines)-1], "确认") {
		t.Fatalf("confirm UI=%q", out.inlines[len(out.inlines)-1])
	}

	if err := ad.HandleCallback(ctx, 1, "cb3", tg.CBConfirm); err != nil {
		t.Fatal(err)
	}
	if len(out.texts) == 0 || !strings.Contains(out.texts[len(out.texts)-1], "已排队") {
		t.Fatalf("queued=%v", out.texts)
	}
}

func TestNotifySendsPhoto(t *testing.T) {
	out := &memOut{}
	ad := tg.New(&botapp.Facade{}, out)
	n := sharedkernel.UserNotify{
		ChatID: 1, TaskID: "t1", Kind: "task_succeeded",
		Outputs: []sharedkernel.BlobRef{{Key: "outputs/t1/0_out.png"}},
	}
	_ = ad.HandleUserNotify(context.Background(), n)
	_ = ad.HandleUserNotify(context.Background(), n)
	if len(out.photos) != 1 {
		t.Fatalf("photos=%v", out.photos)
	}
}
