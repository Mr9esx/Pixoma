package smoke_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/validation"
	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/packaging/botapp"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob/localfs"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance/static"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue/memory"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/application/orchestrator"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/actuator"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/comfyui"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

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
func (m *memCases) List(context.Context, domain.ListQuery) ([]*domain.Case, error) { return nil, nil }
func (m *memCases) Disable(context.Context, sharedkernel.CaseID) error             { return nil }

type memNotify struct{ last *sharedkernel.UserNotify }

func (m *memNotify) Publish(_ context.Context, n sharedkernel.UserNotify) error {
	cp := n
	m.last = &cp
	return nil
}

func TestMemoryAllInOneText2Img(t *testing.T) {
	ctx := context.Background()
	now := time.Unix(1000, 0).UTC()
	bus := memory.New()
	tasks := runtimedomain.NewMemoryTaskRepository()
	n := &memNotify{}
	reg := static.New(instance.Instance{ID: "local", DispatchTopic: "dispatch.local"})
	orch := orchestrator.New(tasks, reg, bus, n)
	orch.Now = func() time.Time { return now }

	store, err := localfs.New(filepath.Join(t.TempDir(), "blob"))
	if err != nil {
		t.Fatal(err)
	}
	worker := &actuator.Worker{
		InstanceID: "local",
		Comfy:      &comfyui.Mock{},
		Blob:       store,
		Status:     bus,
		Ledger:     actuator.NewMemoryLedger(),
		Workflows:  actuator.StaticWorkflows{},
		Now:        func() time.Time { return now },
	}

	_ = bus.Subscribe(ctx, sharedkernel.TopicTaskCreated, func(ctx context.Context, msg queue.Message) error {
		var ev sharedkernel.TaskCreated
		if err := json.Unmarshal(msg.Payload, &ev); err != nil {
			return err
		}
		return orch.OnTaskCreated(ctx, ev)
	})
	_ = bus.Subscribe(ctx, "dispatch.local", func(ctx context.Context, msg queue.Message) error {
		var cmd sharedkernel.DispatchCommand
		if err := json.Unmarshal(msg.Payload, &cmd); err != nil {
			return err
		}
		return worker.HandleDispatch(ctx, cmd)
	})
	_ = bus.Subscribe(ctx, sharedkernel.TopicTaskStatus, func(ctx context.Context, msg queue.Message) error {
		var ev sharedkernel.TaskStatusEvent
		if err := json.Unmarshal(msg.Payload, &ev); err != nil {
			return err
		}
		return orch.OnStatus(ctx, ev)
	})

	doc := domain.CaseDocument{
		ID: "text2img-demo", Name: "Demo",
		Inputs:  []domain.InputField{{Key: "prompt", Type: "string", Required: true}},
		Outputs: []domain.OutputField{{Key: "image", Type: "image"}},
		Bindings: domain.ComfyBindings{
			WorkflowJSON: map[string]any{"1": map[string]any{}},
			Inputs:       []domain.InputBinding{{Key: "prompt", NodeID: "1", FieldPath: "text"}},
		},
		InputSchema: map[string]any{
			"type": "object", "required": []any{"prompt"},
			"properties": map[string]any{"prompt": map[string]any{"type": "string", "minLength": 1}},
		},
	}
	cases := &memCases{}
	_ = cases.Create(ctx, &domain.Case{Document: doc, Enabled: true})

	sessRepo := convdomain.NewMemoryRepository()
	sessSvc := convdomain.NewService(sessRepo, func() sharedkernel.SessionID { return "s1" }, func() time.Time { return now })
	_, _ = sessSvc.StartCase(ctx, 42, "text2img-demo", []string{"prompt"})
	p := "cat"
	_, _ = sessSvc.SubmitInput(ctx, 42, convdomain.DraftValue{Text: &p})

	facade := &botapp.Facade{
		Cases: cases, Validator: validation.New(), Sessions: sessSvc, SessionStore: sessRepo,
		Tasks: tasks, Publisher: bus,
		NewTaskID: func() sharedkernel.TaskID { return "task-smoke" },
		Now:       func() time.Time { return now },
	}
	res, err := facade.ConfirmRun(ctx, botapp.ConfirmRunCmd{ChatID: 42})
	if err != nil {
		t.Fatal(err)
	}
	got, err := tasks.Get(ctx, res.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != sharedkernel.TaskSucceeded {
		t.Fatalf("want succeeded, got %s", got.Status)
	}
	if n.last == nil || n.last.Kind != "task_succeeded" {
		t.Fatalf("notify=%+v", n.last)
	}
}
