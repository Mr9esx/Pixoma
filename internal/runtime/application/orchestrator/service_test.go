package orchestrator_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance/static"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/application/orchestrator"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type memNotify struct {
	items []sharedkernel.UserNotify
}

func (m *memNotify) Publish(_ context.Context, n sharedkernel.UserNotify) error {
	m.items = append(m.items, n)
	return nil
}

type captureBus struct {
	msgs []queue.Message
}

func (c *captureBus) Publish(_ context.Context, msg queue.Message) error {
	c.msgs = append(c.msgs, msg)
	return nil
}

type fakeQuery struct {
	view *orchestrator.ExecutionView
	err  error
}

func (f *fakeQuery) GetRun(context.Context, sharedkernel.TaskID) (*orchestrator.ExecutionView, error) {
	return f.view, f.err
}

func TestOnTaskCreatedDispatches(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(50, 0).UTC()
	_ = tasks.Create(ctx, runtimedomain.NewPending("t1", "s1", "c1", "inputs/t1", now))

	bus := &captureBus{}
	n := &memNotify{}
	reg := static.New(instance.Instance{ID: "local", DispatchTopic: "dispatch.local"})
	svc := orchestrator.New(tasks, reg, bus, n)
	svc.Now = func() time.Time { return now }

	if err := svc.OnTaskCreated(ctx, sharedkernel.TaskCreated{TaskID: "t1"}); err != nil {
		t.Fatal(err)
	}
	got, _ := tasks.Get(ctx, "t1")
	if got.Status != sharedkernel.TaskQueued || got.InstanceID != "local" {
		t.Fatalf("task=%+v", got)
	}
	if len(bus.msgs) != 1 || bus.msgs[0].Topic != "dispatch.local" {
		t.Fatalf("msgs=%+v", bus.msgs)
	}
	var cmd sharedkernel.DispatchCommand
	_ = json.Unmarshal(bus.msgs[0].Payload, &cmd)
	if cmd.TaskID != "t1" {
		t.Fatalf("cmd=%+v", cmd)
	}
}

func TestApplyStatusSucceededIdempotentNotify(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(50, 0).UTC()
	task := runtimedomain.NewPending("t1", "s1", "c1", "inputs/t1", now)
	task.ChatID = 9
	_ = task.MarkQueued("local", now)
	_ = task.MarkRunning("p", now)
	_ = tasks.Create(ctx, task)

	bus := &captureBus{}
	n := &memNotify{}
	svc := orchestrator.New(tasks, static.New(instance.Instance{ID: "local"}), bus, n)

	ev := sharedkernel.TaskStatusEvent{
		TaskID: "t1", Status: sharedkernel.TaskSucceeded,
		Outputs: []sharedkernel.BlobRef{{Key: "out.png"}}, At: now,
	}
	if err := svc.OnStatus(ctx, ev); err != nil {
		t.Fatal(err)
	}
	if err := svc.OnStatus(ctx, ev); err != nil {
		t.Fatal(err)
	}
	if len(n.items) != 1 {
		t.Fatalf("notify count=%d", len(n.items))
	}
}

func TestNotify_JoinsSessionChatID(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(50, 0).UTC()
	task := runtimedomain.NewPending("t1", "s1", "c1", "inputs/t1", now)
	// ChatID left zero — notify must resolve via Session.GetByID
	_ = task.MarkQueued("local", now)
	_ = task.MarkRunning("p", now)
	_ = tasks.Create(ctx, task)

	sessRepo := convdomain.NewMemoryRepository()
	if err := sessRepo.Save(ctx, &convdomain.Session{
		ID:        "s1",
		UserID:    "u1",
		ChatID:    100,
		CaseID:    "c1",
		Status:    convdomain.StatusSubmitted,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	n := &memNotify{}
	svc := orchestrator.New(tasks, static.New(instance.Instance{ID: "local"}), &captureBus{}, n)
	svc.Sessions = sessRepo

	ev := sharedkernel.TaskStatusEvent{
		TaskID: "t1", Status: sharedkernel.TaskSucceeded,
		Outputs: []sharedkernel.BlobRef{{Key: "out.png"}}, At: now,
	}
	if err := svc.OnStatus(ctx, ev); err != nil {
		t.Fatal(err)
	}
	if len(n.items) != 1 {
		t.Fatalf("notify count=%d", len(n.items))
	}
	if n.items[0].ChatID != 100 {
		t.Fatalf("chat_id=%d want 100 (via session join)", n.items[0].ChatID)
	}
}

func TestCancelPending(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(50, 0).UTC()
	_ = tasks.Create(ctx, runtimedomain.NewPending("t1", "s1", "c1", "inputs/t1", now))
	n := &memNotify{}
	svc := orchestrator.New(tasks, static.New(), &captureBus{}, n)
	svc.Now = func() time.Time { return now }
	if err := svc.RequestCancel(ctx, "t1"); err != nil {
		t.Fatal(err)
	}
	got, _ := tasks.Get(ctx, "t1")
	if got.Status != sharedkernel.TaskCancelled {
		t.Fatal(got.Status)
	}
}

func TestReconcileSucceeded(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(100, 0).UTC()
	task := runtimedomain.NewPending("t1", "s1", "c1", "inputs/t1", now.Add(-2*time.Minute))
	_ = task.MarkQueued("local", now.Add(-2*time.Minute))
	_ = task.MarkRunning("p", now.Add(-2*time.Minute))
	_ = tasks.Create(ctx, task)

	n := &memNotify{}
	svc := orchestrator.New(tasks, static.New(instance.Instance{ID: "local"}), &captureBus{}, n)
	svc.Now = func() time.Time { return now }
	svc.Query = &fakeQuery{view: &orchestrator.ExecutionView{
		TaskID: "t1", Phase: "succeeded", Outputs: []sharedkernel.BlobRef{{Key: "x.png"}},
	}}
	if err := svc.ReconcileStale(ctx, time.Minute, 10); err != nil {
		t.Fatal(err)
	}
	got, _ := tasks.Get(ctx, "t1")
	if got.Status != sharedkernel.TaskSucceeded {
		t.Fatal(got.Status)
	}
}

func TestCircuitBreakerOpens(t *testing.T) {
	cb := orchestrator.NewCircuitBreaker(2, time.Minute)
	cb.RecordFailure("a")
	if !cb.Allow("a") {
		t.Fatal("should still allow")
	}
	cb.RecordFailure("a")
	if cb.Allow("a") {
		t.Fatal("should be open")
	}
}
