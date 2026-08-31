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

func TestSplit_TopicClaimIsolation(t *testing.T) {
	ctx := context.Background()
	now := time.Unix(1000, 0).UTC()
	bus := memory.New()
	tasks := runtimedomain.NewMemoryTaskRepository()
	n := &memNotify{}
	reg := static.New(
		edge.Instance{ID: "gpu-a", SubscribeTopics: []string{"default"}},
		edge.Instance{ID: "gpu-b", SubscribeTopics: []string{"fast-gpu"}},
	)
	orch := orchestrator.New(tasks, reg, bus, n)
	orch.Now = func() time.Time { return now }

	cases := &memCases{}
	cases.Create(ctx, &domain.Case{Document: domain.CaseDocument{
		ID:     8,
		Name:   "Split",
		Inputs: []domain.InputField{{Key: "prompt", Type: "string", Required: true}},
		Routing: &domain.RoutingConfig{Rules: []domain.RoutingRule{
			{When: json.RawMessage(`{"field":"user.is_premium","op":"eq","value":true}`), Topic: "fast-gpu"},
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
	_ = sessions.Save(ctx, &convdomain.Session{ID: "s1", UserID: "u1", ChatID: "tg:1", CaseID: 8})
	orch.Sessions = sessions
	orch.Cases = caseDocReader{cases: cases}
	cond := condition.NewRegistry()
	cond.Register(&condition.UserProvider{Lookup: func(context.Context, string) (*bool, error) {
		trueVal := true
		return &trueVal, nil
	}})
	orch.Condition = cond

	store, err := localfs.New(filepath.Join(t.TempDir(), "blob"))
	if err != nil {
		t.Fatal(err)
	}
	mock := &comfyuitest.Fake{SubmitFn: func(_ context.Context, _ comfyui.Graph) (string, error) {
		return "prompt-split", nil
	}}
	snap := &actuator.CaseSnapshot{Tasks: tasks, Cases: cases, Blob: store, Uploader: mock}
	orch.Prep = snap
	workerB := &actuator.Worker{
		EdgeID:    "gpu-b",
		Comfy:     mock,
		Blob:      store,
		Status:    bus,
		Workflows: snap,
		Now:       func() time.Time { return now },
	}

	_ = bus.Subscribe(ctx, sharedkernel.TopicTaskCreated, func(ctx context.Context, msg queue.Message) error {
		var ev sharedkernel.TaskCreated
		if err := json.Unmarshal(msg.Payload, &ev); err != nil {
			return err
		}
		if err := orch.OnTaskCreated(ctx, ev); err != nil {
			return err
		}
		// Edge A only subscribes default; Edge B only fast-gpu.
		claimedA, err := tasks.ClaimNextWithLease(ctx, "gpu-a", []string{"default"}, 90*time.Second, now)
		if err != nil {
			return err
		}
		if claimedA != nil {
			return workerB.HandleDispatch(ctx, sharedkernel.DispatchCommand{
				TaskID: claimedA.ID, EdgeID: claimedA.EdgeID, JobRef: claimedA.JobRef,
			})
		}
		claimedB, err := tasks.ClaimNextWithLease(ctx, "gpu-b", []string{"fast-gpu"}, 90*time.Second, now)
		if err != nil {
			return err
		}
		if claimedB != nil {
			return workerB.HandleDispatch(ctx, sharedkernel.DispatchCommand{
				TaskID: claimedB.ID, EdgeID: claimedB.EdgeID, JobRef: claimedB.JobRef,
			})
		}
		return nil
	})
	_ = bus.Subscribe(ctx, sharedkernel.TopicTaskStatus, func(ctx context.Context, msg queue.Message) error {
		var ev sharedkernel.TaskStatusEvent
		if err := json.Unmarshal(msg.Payload, &ev); err != nil {
			return err
		}
		return orch.OnStatus(ctx, ev)
	})

	task := runtimedomain.NewPending("task-split", "s1", sharedkernel.CaseID(8), "inputs/task-split", now)
	_ = tasks.Create(ctx, task)
	if _, err := store.Put(ctx, "inputs/task-split/prompt.txt", strings.NewReader("hi"), blob.PutOptions{MIME: "text/plain"}); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(sharedkernel.TaskCreated{TaskID: "task-split", CreatedAt: now})
	if err := bus.Publish(ctx, queue.Message{Topic: sharedkernel.TopicTaskCreated, Key: "task-split", Payload: payload}); err != nil {
		t.Fatal(err)
	}

	got, _ := tasks.Get(ctx, "task-split")
	if got.Status != sharedkernel.TaskSucceeded {
		t.Fatalf("status = %s", got.Status)
	}
	if got.DispatchTopic != "fast-gpu" || got.EdgeID != "gpu-b" {
		t.Fatalf("topic=%q edge=%q, want fast-gpu/gpu-b (edge A must not claim)", got.DispatchTopic, got.EdgeID)
	}
}
