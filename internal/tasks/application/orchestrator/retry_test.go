package orchestrator_test

import (
	"context"
	"testing"
	"time"

	edge "github.com/Mr9esx/Pixoma/internal/edge/domain"
	"github.com/Mr9esx/Pixoma/internal/edge/infrastructure/static"
	"github.com/Mr9esx/Pixoma/internal/tasks/application/orchestrator"
	runtimedomain "github.com/Mr9esx/Pixoma/internal/tasks/domain"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

func TestApplyStatus_FailureRequeuesWithBackoff(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(200, 0).UTC()
	task := runtimedomain.NewPending("t1", "s1", sharedkernel.CaseID(1), "in", now)
	_ = task.PrepareForTopic("fast-gpu", sharedkernel.BlobRef{Key: "j"}, now)
	_ = task.ClaimWithLease("gpu-1", time.Minute, now)
	_ = tasks.Create(ctx, task)

	n := &memNotify{}
	reg := static.New(edge.Instance{ID: "gpu-1", SubscribeTopics: []string{"fast-gpu"}})
	svc := orchestrator.New(tasks, reg, &captureBus{}, n)
	svc.Now = func() time.Time { return now }

	if err := svc.OnStatus(ctx, sharedkernel.TaskStatusEvent{TaskID: "t1", Status: sharedkernel.TaskFailed, At: now, ErrorMsg: "boom"}); err != nil {
		t.Fatalf("status: %v", err)
	}
	got, _ := tasks.Get(ctx, "t1")
	if got.Status != sharedkernel.TaskQueued || got.Attempts != 1 {
		t.Fatalf("after failure: %+v", got)
	}
	if got.DispatchTopic != "fast-gpu" || got.EdgeID != "" {
		t.Fatalf("topic/edge after failure: topic=%q edge=%q", got.DispatchTopic, got.EdgeID)
	}
	if got.RequeueAt.Sub(now) != 5*time.Second {
		t.Fatalf("requeue_at = %v", got.RequeueAt.Sub(now))
	}
	if len(n.items) != 0 {
		t.Fatalf("no notify expected on retry, got %d", len(n.items))
	}
}

func TestApplyStatus_ExhaustedRetriesFinalFailed(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(200, 0).UTC()
	task := runtimedomain.NewPending("t2", "s1", sharedkernel.CaseID(1), "in", now)
	_ = task.PrepareForTopic("default", sharedkernel.BlobRef{Key: "j"}, now)
	task.Attempts = runtimedomain.MaxRetries
	_ = task.ClaimWithLease("gpu-1", time.Minute, now)
	_ = tasks.Create(ctx, task)

	n := &memNotify{}
	svc := orchestrator.New(tasks, static.New(), &captureBus{}, n)
	svc.Now = func() time.Time { return now }

	if err := svc.OnStatus(ctx, sharedkernel.TaskStatusEvent{TaskID: "t2", Status: sharedkernel.TaskFailed, At: now, ErrorMsg: "last"}); err != nil {
		t.Fatalf("status: %v", err)
	}
	got, _ := tasks.Get(ctx, "t2")
	if got.Status != sharedkernel.TaskFailed {
		t.Fatalf("status = %s, want failed", got.Status)
	}
	if got.ErrorCode != "max_retries" {
		t.Fatalf("error_code = %q", got.ErrorCode)
	}
	if len(n.items) != 1 {
		t.Fatalf("expected one notify, got %d", len(n.items))
	}

	// Repeated failed status is idempotent.
	if err := svc.OnStatus(ctx, sharedkernel.TaskStatusEvent{TaskID: "t2", Status: sharedkernel.TaskFailed, At: now, ErrorMsg: "again"}); err != nil {
		t.Fatalf("repeat status: %v", err)
	}
	got, _ = tasks.Get(ctx, "t2")
	if got.Attempts != runtimedomain.MaxRetries+1 {
		t.Fatalf("attempts changed on repeat: %d", got.Attempts)
	}
	if len(n.items) != 1 {
		t.Fatalf("duplicate notify: %d", len(n.items))
	}
}
