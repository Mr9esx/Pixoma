package orchestrator_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance/static"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/notify"
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

func TestRequestCancelViaOrchestrator(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(50, 0).UTC()
	_ = tasks.Create(ctx, runtimedomain.NewPending("t1", "s1", "c1", "inputs/t1", now))

	orch := orchestrator.New(tasks, nil, nil, notify.Nop{})
	orch.Now = func() time.Time { return now }
	if err := orch.RequestCancel(ctx, "t1"); err != nil {
		t.Fatal(err)
	}
	got, err := tasks.Get(ctx, "t1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != sharedkernel.TaskCancelled {
		t.Fatalf("status=%s", got.Status)
	}

	done := runtimedomain.NewPending("t2", "s1", "c1", "inputs/t2", now)
	if err := done.MarkQueued("local", now); err != nil {
		t.Fatal(err)
	}
	if err := done.MarkRunning("p1", now); err != nil {
		t.Fatal(err)
	}
	if err := done.MarkSucceeded(nil, now); err != nil {
		t.Fatal(err)
	}
	if err := tasks.Create(ctx, done); err != nil {
		t.Fatal(err)
	}
	if err := orch.RequestCancel(ctx, "t2"); !errors.Is(err, runtimedomain.ErrCancelNotAllowed) {
		t.Fatalf("want ErrCancelNotAllowed, got %v", err)
	}
}

func TestReconcileReadsTaskOnly(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(100, 0).UTC()
	staleAt := now.Add(-2 * time.Minute)
	task := runtimedomain.NewPending("t1", "s1", "c1", "inputs/t1", staleAt)
	_ = task.MarkQueued("local", staleAt)
	_ = task.MarkRunning("prompt-keep", staleAt)
	_ = tasks.Create(ctx, task)

	n := &memNotify{}
	svc := orchestrator.New(tasks, static.New(instance.Instance{ID: "local"}), &captureBus{}, n)
	svc.Now = func() time.Time { return now }
	// No Query / Ledger: reconcile must use Tasks.Get only.
	// Already-running with no new execution info must not bump UpdatedAt.
	if err := svc.ReconcileStale(ctx, time.Minute, 10); err != nil {
		t.Fatal(err)
	}
	got, _ := tasks.Get(ctx, "t1")
	if got.Status != sharedkernel.TaskRunning {
		t.Fatalf("status=%s", got.Status)
	}
	if got.PromptID != "prompt-keep" {
		t.Fatalf("prompt=%s", got.PromptID)
	}
	if !got.UpdatedAt.Equal(staleAt) {
		t.Fatalf("updated_at=%v want stale %v (no no-op refresh)", got.UpdatedAt, staleAt)
	}
}

func TestReconcile_StaleQueuedWithoutPromptRePend(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(100, 0).UTC()
	staleAt := now.Add(-2 * time.Minute)
	task := runtimedomain.NewPending("t-q", "s1", "c1", "inputs/t-q", staleAt)
	_ = task.MarkQueued("local", staleAt)
	_ = tasks.Create(ctx, task)

	svc := orchestrator.New(tasks, static.New(instance.Instance{ID: "local"}), &captureBus{}, &memNotify{})
	svc.Now = func() time.Time { return now }
	if err := svc.ReconcileStale(ctx, time.Minute, 10); err != nil {
		t.Fatal(err)
	}
	got, _ := tasks.Get(ctx, "t-q")
	if got.Status != sharedkernel.TaskPending {
		t.Fatalf("status=%s want pending", got.Status)
	}
	if got.InstanceID != "" {
		t.Fatalf("instance_id=%q want empty", got.InstanceID)
	}
}

type failPublishBus struct {
	err error
}

func (f *failPublishBus) Publish(_ context.Context, _ queue.Message) error {
	return f.err
}

func TestDispatch_PublishFailRollsBackPending(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(50, 0).UTC()
	_ = tasks.Create(ctx, runtimedomain.NewPending("t1", "s1", "c1", "inputs/t1", now))

	bus := &failPublishBus{err: errors.New("bus down")}
	svc := orchestrator.New(tasks, static.New(instance.Instance{ID: "local", DispatchTopic: "dispatch.local"}), bus, &memNotify{})
	svc.Now = func() time.Time { return now }

	err := svc.OnTaskCreated(ctx, sharedkernel.TaskCreated{TaskID: "t1"})
	if err == nil {
		t.Fatal("expected publish error")
	}
	got, _ := tasks.Get(ctx, "t1")
	if got.Status != sharedkernel.TaskPending {
		t.Fatalf("status=%s want pending after publish fail", got.Status)
	}
	if got.InstanceID != "" {
		t.Fatalf("instance_id=%q want empty", got.InstanceID)
	}
}

type countingBus struct {
	mu   sync.Mutex
	msgs []queue.Message
}

func (c *countingBus) Publish(_ context.Context, msg queue.Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.msgs = append(c.msgs, msg)
	return nil
}

func (c *countingBus) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.msgs)
}

func TestDispatch_ConcurrentClaimOnlyOnePublishes(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(50, 0).UTC()
	_ = tasks.Create(ctx, runtimedomain.NewPending("t1", "s1", "c1", "inputs/t1", now))

	bus := &countingBus{}
	reg := static.New(
		instance.Instance{ID: "gpu-a", DispatchTopic: "dispatch.gpu-a"},
		instance.Instance{ID: "gpu-b", DispatchTopic: "dispatch.gpu-b"},
	)
	svc := orchestrator.New(tasks, reg, bus, &memNotify{})
	svc.Now = func() time.Time { return now }

	const n = 32
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_ = svc.OnTaskCreated(ctx, sharedkernel.TaskCreated{TaskID: "t1"})
		}()
	}
	wg.Wait()

	if bus.len() != 1 {
		t.Fatalf("publishes=%d want 1", bus.len())
	}
	got, _ := tasks.Get(ctx, "t1")
	if got.Status != sharedkernel.TaskQueued {
		t.Fatalf("status=%s want queued", got.Status)
	}
	if got.InstanceID != "gpu-a" && got.InstanceID != "gpu-b" {
		t.Fatalf("instance_id=%q", got.InstanceID)
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

func TestOrchestrator_RoundRobinAcrossHealthy(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(50, 0).UTC()
	_ = tasks.Create(ctx, runtimedomain.NewPending("t1", "s1", "c1", "inputs/t1", now))
	_ = tasks.Create(ctx, runtimedomain.NewPending("t2", "s1", "c1", "inputs/t2", now))

	bus := &captureBus{}
	reg := static.New(
		instance.Instance{ID: "gpu-a", DispatchTopic: "dispatch.gpu-a"},
		instance.Instance{ID: "gpu-b", DispatchTopic: "dispatch.gpu-b"},
	)
	svc := orchestrator.New(tasks, reg, bus, &memNotify{})
	svc.Now = func() time.Time { return now }

	if err := svc.OnTaskCreated(ctx, sharedkernel.TaskCreated{TaskID: "t1"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.OnTaskCreated(ctx, sharedkernel.TaskCreated{TaskID: "t2"}); err != nil {
		t.Fatal(err)
	}

	t1, _ := tasks.Get(ctx, "t1")
	t2, _ := tasks.Get(ctx, "t2")
	if t1.InstanceID != "gpu-a" {
		t.Fatalf("t1 instance=%s want gpu-a", t1.InstanceID)
	}
	if t2.InstanceID != "gpu-b" {
		t.Fatalf("t2 instance=%s want gpu-b", t2.InstanceID)
	}
}

func TestOrchestrator_NoInstanceKeepsPending(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(50, 0).UTC()
	_ = tasks.Create(ctx, runtimedomain.NewPending("t1", "s1", "c1", "inputs/t1", now))

	svc := orchestrator.New(tasks, static.New(), &captureBus{}, &memNotify{})
	svc.Now = func() time.Time { return now }

	if err := svc.SchedulePending(ctx, 10); err != nil {
		t.Fatalf("SchedulePending must not fail when no instance: %v", err)
	}
	got, _ := tasks.Get(ctx, "t1")
	if got.Status != sharedkernel.TaskPending {
		t.Fatalf("status=%s want pending", got.Status)
	}
	if got.InstanceID != "" {
		t.Fatalf("instance_id=%q want empty", got.InstanceID)
	}
}
