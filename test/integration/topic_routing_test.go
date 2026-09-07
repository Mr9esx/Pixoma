package smoke_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob/localfs"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge/static"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue/memory"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/application/orchestrator"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/domain/condition"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/actuator"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/comfyui"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/comfyui/comfyuitest"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type caseDocReader struct{ cases *memCases }

func (c caseDocReader) GetCase(_ context.Context, id sharedkernel.CaseID) (*domain.CaseDocument, error) {
	got, err := c.cases.Get(context.Background(), id)
	if err != nil {
		return nil, err
	}
	return &got.Document, nil
}

func TestMemoryAllInOne_TopicRoutedToFastGpu(t *testing.T) {
	ctx := context.Background()
	now := time.Unix(1000, 0).UTC()
	bus := memory.New()
	tasks := runtimedomain.NewMemoryTaskRepository()
	n := &memNotify{}
	reg := static.New(edge.Instance{ID: "local", SubscribeTopics: []string{"default", "fast-gpu"}})
	orch := orchestrator.New(tasks, reg, bus, n)
	orch.Now = func() time.Time { return now }

	cases := &memCases{}
	cases.Create(ctx, &domain.Case{Document: domain.CaseDocument{
		ID:     7,
		Name:   "Routed",
		Inputs: []domain.InputField{{Key: "prompt", Type: "string", Required: true}},
		Routing: &domain.RoutingConfig{Rules: []domain.RoutingRule{
			{When: json.RawMessage(`{"field":"user.level","op":"eq","value":"image"}`), Topic: "fast-gpu"},
		}},
		Bindings: domain.ComfyBindings{
			WorkflowJSON: map[string]any{
				"1": map[string]any{"class_type": "CLIPTextEncode", "inputs": map[string]any{"text": "x"}},
			},
			Inputs: []domain.InputBinding{{Key: "prompt", NodeID: "1", FieldPath: "text"}},
		},
		InputSchema: map[string]any{
			"type": "object", "required": []any{"prompt"},
			"properties": map[string]any{"prompt": map[string]any{"type": "string", "minLength": 1}},
		},
	}, Enabled: true})

	sessions := convdomain.NewMemoryRepository()
	_ = sessions.Save(ctx, &convdomain.Session{ID: "s1", UserID: "u1", ChatID: "tg:42", CaseID: 7})
	orch.Sessions = sessions
	orch.Cases = caseDocReader{cases: cases}
	cond := condition.NewRegistry()
	cond.Register(&smokeStubProvider{val: func(context.Context, string) (any, error) {
		return "image", nil
	}})
	orch.Condition = cond

	store, err := localfs.New(filepath.Join(t.TempDir(), "blob"))
	if err != nil {
		t.Fatal(err)
	}
	mock := &comfyuitest.Fake{SubmitFn: func(_ context.Context, _ comfyui.Graph) (string, error) {
		return "prompt-routed", nil
	}}
	snap := &actuator.CaseSnapshot{Tasks: tasks, Cases: cases, Blob: store, Uploader: mock}
	orch.Prep = snap
	worker := &actuator.Worker{
		EdgeID:    "local",
		Comfy:     mock,
		Blob:      store,
		Status: bus,
		Now:    func() time.Time { return now },
	}

	_ = bus.Subscribe(ctx, sharedkernel.TopicTaskCreated, func(ctx context.Context, msg queue.Message) error {
		var ev sharedkernel.TaskCreated
		if err := json.Unmarshal(msg.Payload, &ev); err != nil {
			return err
		}
		if err := orch.OnTaskCreated(ctx, ev); err != nil {
			return err
		}
		claimed, err := tasks.ClaimNextWithLease(ctx, "local", []string{"fast-gpu", "default"}, 90*time.Second, now)
		if err != nil {
			return err
		}
		if claimed == nil {
			return nil
		}
		return worker.HandleDispatch(ctx, sharedkernel.DispatchCommand{
			TaskID: claimed.ID, EdgeID: claimed.EdgeID, JobRef: claimed.JobRef,
		})
	})
	_ = bus.Subscribe(ctx, sharedkernel.TopicTaskStatus, func(ctx context.Context, msg queue.Message) error {
		var ev sharedkernel.TaskStatusEvent
		if err := json.Unmarshal(msg.Payload, &ev); err != nil {
			return err
		}
		return orch.OnStatus(ctx, ev)
	})

	task := runtimedomain.NewPending("task-routed", "s1", sharedkernel.CaseID(7), "inputs/task-routed", now)
	_ = tasks.Create(ctx, task)
	if _, err := store.Put(ctx, "inputs/task-routed/prompt.txt", strings.NewReader("a cat"), blob.PutOptions{MIME: "text/plain"}); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(sharedkernel.TaskCreated{TaskID: "task-routed", CreatedAt: now})
	if err := bus.Publish(ctx, queue.Message{Topic: sharedkernel.TopicTaskCreated, Key: "task-routed", Payload: payload}); err != nil {
		t.Fatal(err)
	}

	got, err := tasks.Get(ctx, "task-routed")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != sharedkernel.TaskSucceeded {
		t.Fatalf("status = %s, want succeeded", got.Status)
	}
	if got.DispatchTopic != "fast-gpu" {
		t.Fatalf("dispatch_topic = %q, want fast-gpu", got.DispatchTopic)
	}
	if n.last == nil {
		t.Fatal("expected notify")
	}
}
