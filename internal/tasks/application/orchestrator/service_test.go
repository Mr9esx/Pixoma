package orchestrator_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	catalogdomain "github.com/Mr9esx/Pixoma/internal/cases/domain"
	convdomain "github.com/Mr9esx/Pixoma/internal/sessions/domain"
	edge "github.com/Mr9esx/Pixoma/internal/edge/domain"
	"github.com/Mr9esx/Pixoma/internal/edge/infrastructure/static"
	"github.com/Mr9esx/Pixoma/internal/platform/notify"
	"github.com/Mr9esx/Pixoma/internal/platform/queue"
	"github.com/Mr9esx/Pixoma/internal/tasks/application/orchestrator"
	runtimedomain "github.com/Mr9esx/Pixoma/internal/tasks/domain"
	"github.com/Mr9esx/Pixoma/internal/tasks/domain/condition"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
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

type stubPrep struct{}

func setExplicitDefaultRouting(svc *orchestrator.Service) {
	svc.Cases = &caseReaderStub{docs: map[sharedkernel.CaseID]*catalogdomain.CaseDocument{
		1: {
			ID:      1,
			Routing: &catalogdomain.RoutingConfig{Rules: []catalogdomain.RoutingRule{{When: []byte(`{"always":true}`), Topic: "default"}}},
		},
	}}
	svc.Condition = condition.NewRegistry()
}

func (stubPrep) PrepareJob(_ context.Context, taskID sharedkernel.TaskID) (sharedkernel.BlobRef, error) {
	return sharedkernel.BlobRef{Key: "jobs/" + string(taskID) + "/job.json", MIME: "application/json"}, nil
}

type failPrep struct{ err error }

func (f failPrep) PrepareJob(_ context.Context, _ sharedkernel.TaskID) (sharedkernel.BlobRef, error) {
	return sharedkernel.BlobRef{}, f.err
}

func TestOnTaskCreatedMakesClaimable(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(50, 0).UTC()
	_ = tasks.Create(ctx, runtimedomain.NewPending("t1", "s1", sharedkernel.CaseID(1), "inputs/t1", now))

	bus := &captureBus{}
	n := &memNotify{}
	reg := static.New(edge.Instance{ID: "local", SubscribeTopics: []string{"default"}})
	svc := orchestrator.New(tasks, reg, bus, n)
	svc.Now = func() time.Time { return now }
	svc.Prep = stubPrep{}
	setExplicitDefaultRouting(svc)

	if err := svc.OnTaskCreated(ctx, sharedkernel.TaskCreated{TaskID: "t1"}); err != nil {
		t.Fatal(err)
	}
	got, _ := tasks.Get(ctx, "t1")
	if got.Status != sharedkernel.TaskQueued || got.EdgeID != "" {
		t.Fatalf("task=%+v", got)
	}
	if got.DispatchTopic != "default" {
		t.Fatalf("dispatch_topic=%q, want default", got.DispatchTopic)
	}
	if got.JobRef.Key != "jobs/t1/job.json" {
		t.Fatalf("job_ref=%+v", got.JobRef)
	}
	if len(bus.msgs) != 0 {
		t.Fatalf("must not publish redis/memory dispatch, msgs=%+v", bus.msgs)
	}
}

func TestApplyStatusSucceededIdempotentNotify(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(50, 0).UTC()
	task := runtimedomain.NewPending("t1", "s1", sharedkernel.CaseID(1), "inputs/t1", now)
	task.ChatID = "tg:9"
	_ = task.MarkQueued("local", now)
	_ = task.MarkRunning("p", now)
	_ = tasks.Create(ctx, task)

	bus := &captureBus{}
	n := &memNotify{}
	svc := orchestrator.New(tasks, static.New(edge.Instance{ID: "local"}), bus, n)

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

func TestApplyStatusRejectsWrongInstance(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(50, 0).UTC()
	task := runtimedomain.NewPending("t1", "s1", sharedkernel.CaseID(1), "inputs/t1", now)
	_ = task.PrepareForClaim("gpu-2", sharedkernel.BlobRef{Key: "j"}, now)
	_ = task.ClaimWithLease("gpu-2", time.Minute, now)
	if err := tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}

	svc := orchestrator.New(tasks, static.New(edge.Instance{ID: "gpu-2"}), &captureBus{}, &memNotify{})
	svc.Now = func() time.Time { return now }
	err := svc.OnStatus(ctx, sharedkernel.TaskStatusEvent{
		TaskID: "t1", EdgeID: "gpu-1", Status: sharedkernel.TaskSucceeded, At: now,
	})
	if !errors.Is(err, orchestrator.ErrStaleHolder) {
		t.Fatalf("got %v", err)
	}
	got, _ := tasks.Get(ctx, "t1")
	if got.Status != sharedkernel.TaskRunning {
		t.Fatalf("status=%s", got.Status)
	}
}

func TestNotify_JoinsSessionChatID(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(50, 0).UTC()
	task := runtimedomain.NewPending("t1", "s1", sharedkernel.CaseID(1), "inputs/t1", now)
	_ = task.MarkQueued("local", now)
	_ = task.MarkRunning("p", now)
	_ = tasks.Create(ctx, task)

	sessRepo := convdomain.NewMemoryRepository()
	if err := sessRepo.Save(ctx, &convdomain.Session{
		ID:        "s1",
		UserID:    "u1",
		ChatID:    "tg:100",
		CaseID:    1,
		Status:    convdomain.StatusSubmitted,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	n := &memNotify{}
	svc := orchestrator.New(tasks, static.New(edge.Instance{ID: "local"}), &captureBus{}, n)
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
	if n.items[0].ChatID != "tg:100" {
		t.Fatalf("chat_id=%s want tg:100 (via session join)", n.items[0].ChatID)
	}
}

func TestCancelPending(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(50, 0).UTC()
	_ = tasks.Create(ctx, runtimedomain.NewPending("t1", "s1", sharedkernel.CaseID(1), "inputs/t1", now))
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
	_ = tasks.Create(ctx, runtimedomain.NewPending("t1", "s1", sharedkernel.CaseID(1), "inputs/t1", now))

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

	done := runtimedomain.NewPending("t2", "s1", sharedkernel.CaseID(1), "inputs/t2", now)
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
	task := runtimedomain.NewPending("t1", "s1", sharedkernel.CaseID(1), "inputs/t1", staleAt)
	_ = task.MarkQueued("local", staleAt)
	_ = task.MarkRunning("prompt-keep", staleAt)
	_ = tasks.Create(ctx, task)

	n := &memNotify{}
	svc := orchestrator.New(tasks, static.New(edge.Instance{ID: "local"}), &captureBus{}, n)
	svc.Now = func() time.Time { return now }
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
	task := runtimedomain.NewPending("t-q", "s1", sharedkernel.CaseID(1), "inputs/t-q", staleAt)
	_ = task.MarkQueued("local", staleAt)
	_ = tasks.Create(ctx, task)

	svc := orchestrator.New(tasks, static.New(edge.Instance{ID: "local"}), &captureBus{}, &memNotify{})
	svc.Now = func() time.Time { return now }
	if err := svc.ReconcileStale(ctx, time.Minute, 10); err != nil {
		t.Fatal(err)
	}
	got, _ := tasks.Get(ctx, "t-q")
	if got.Status != sharedkernel.TaskPending {
		t.Fatalf("status=%s want pending", got.Status)
	}
	if got.EdgeID != "" {
		t.Fatalf("instance_id=%q want empty", got.EdgeID)
	}
}

type missingCaseReader struct{}

func (missingCaseReader) GetCase(context.Context, sharedkernel.CaseID) (*catalogdomain.CaseDocument, error) {
	return nil, catalogdomain.ErrNotFound
}

func TestDispatchPendingFailsTerminalWhenCaseDeleted(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(60, 0).UTC()
	task := runtimedomain.NewPending("t1", "s1", sharedkernel.CaseID(1), "inputs/t1", now)
	task.ChatID = "tg:9"
	_ = tasks.Create(ctx, task)
	n := &memNotify{}
	svc := orchestrator.New(tasks, static.New(edge.Instance{ID: "local", DispatchTopic: "dispatch.local"}), &captureBus{}, n)
	svc.Now = func() time.Time { return now }
	svc.Cases = missingCaseReader{}
	svc.Condition = condition.NewRegistry()

	if err := svc.SchedulePending(ctx, 1); err != nil {
		t.Fatal(err)
	}
	got, err := tasks.Get(ctx, "t1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != sharedkernel.TaskFailed || got.ErrorCode != sharedkernel.TaskErrorCaseDeleted {
		t.Fatalf("task=%+v", got)
	}
	if len(n.items) != 1 || n.items[0].Kind != "task_failed" {
		t.Fatalf("notifies=%+v", n.items)
	}
}

func TestDispatch_PrepFailKeepsPending(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(50, 0).UTC()
	_ = tasks.Create(ctx, runtimedomain.NewPending("t1", "s1", sharedkernel.CaseID(1), "inputs/t1", now))

	svc := orchestrator.New(tasks, static.New(edge.Instance{ID: "local", SubscribeTopics: []string{"default"}}), &captureBus{}, &memNotify{})
	svc.Now = func() time.Time { return now }
	svc.Prep = failPrep{err: errors.New("blob down")}
	setExplicitDefaultRouting(svc)

	err := svc.OnTaskCreated(ctx, sharedkernel.TaskCreated{TaskID: "t1"})
	if err == nil {
		t.Fatal("expected prep error")
	}
	got, _ := tasks.Get(ctx, "t1")
	if got.Status != sharedkernel.TaskPending {
		t.Fatalf("status=%s want pending after prep fail", got.Status)
	}
	if got.EdgeID != "" {
		t.Fatalf("instance_id=%q want empty", got.EdgeID)
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

func TestDispatch_ConcurrentPrepareOnlyOneClaimable(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(50, 0).UTC()
	_ = tasks.Create(ctx, runtimedomain.NewPending("t1", "s1", sharedkernel.CaseID(1), "inputs/t1", now))

	bus := &countingBus{}
	reg := static.New(
		edge.Instance{ID: "gpu-a", SubscribeTopics: []string{"default"}},
		edge.Instance{ID: "gpu-b", SubscribeTopics: []string{"default"}},
	)
	svc := orchestrator.New(tasks, reg, bus, &memNotify{})
	svc.Now = func() time.Time { return now }
	svc.Prep = stubPrep{}
	setExplicitDefaultRouting(svc)

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

	if bus.len() != 0 {
		t.Fatalf("publishes=%d want 0", bus.len())
	}
	got, _ := tasks.Get(ctx, "t1")
	if got.Status != sharedkernel.TaskQueued {
		t.Fatalf("status=%s want queued", got.Status)
	}
	if got.EdgeID != "" || got.DispatchTopic != "default" {
		t.Fatalf("edge=%q topic=%q want unbound default", got.EdgeID, got.DispatchTopic)
	}
	if got.JobRef.Key == "" {
		t.Fatal("expected job_ref")
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

func TestOrchestrator_TopicClaimableWithoutRoundRobin(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(50, 0).UTC()
	_ = tasks.Create(ctx, runtimedomain.NewPending("t1", "s1", sharedkernel.CaseID(1), "inputs/t1", now))
	_ = tasks.Create(ctx, runtimedomain.NewPending("t2", "s1", sharedkernel.CaseID(1), "inputs/t2", now))

	bus := &captureBus{}
	reg := static.New(
		edge.Instance{ID: "gpu-a", SubscribeTopics: []string{"default"}},
		edge.Instance{ID: "gpu-b", SubscribeTopics: []string{"default"}},
	)
	svc := orchestrator.New(tasks, reg, bus, &memNotify{})
	svc.Now = func() time.Time { return now }
	svc.Prep = stubPrep{}
	setExplicitDefaultRouting(svc)

	if err := svc.OnTaskCreated(ctx, sharedkernel.TaskCreated{TaskID: "t1"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.OnTaskCreated(ctx, sharedkernel.TaskCreated{TaskID: "t2"}); err != nil {
		t.Fatal(err)
	}

	t1, _ := tasks.Get(ctx, "t1")
	t2, _ := tasks.Get(ctx, "t2")
	if t1.EdgeID != "" || t1.DispatchTopic != "default" {
		t.Fatalf("t1 edge=%q topic=%q want unbound default", t1.EdgeID, t1.DispatchTopic)
	}
	if t2.EdgeID != "" || t2.DispatchTopic != "default" {
		t.Fatalf("t2 edge=%q topic=%q want unbound default", t2.EdgeID, t2.DispatchTopic)
	}
}

func TestOrchestrator_NoInstanceKeepsPending(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(50, 0).UTC()
	_ = tasks.Create(ctx, runtimedomain.NewPending("t1", "s1", sharedkernel.CaseID(1), "inputs/t1", now))

	svc := orchestrator.New(tasks, static.New(), &captureBus{}, &memNotify{})
	svc.Now = func() time.Time { return now }
	svc.Prep = stubPrep{}
	setExplicitDefaultRouting(svc)

	if err := svc.SchedulePending(ctx, 10); err != nil {
		t.Fatalf("SchedulePending must not fail when no instance: %v", err)
	}
	got, _ := tasks.Get(ctx, "t1")
	if got.Status != sharedkernel.TaskPending {
		t.Fatalf("status=%s want pending", got.Status)
	}
	if got.EdgeID != "" {
		t.Fatalf("instance_id=%q want empty", got.EdgeID)
	}
}
