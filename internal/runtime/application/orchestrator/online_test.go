package orchestrator_test

import (
	"context"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance/static"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/application/orchestrator"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestDispatchSkippedWhenOnlineFilterRejects(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(1, 0).UTC()
	if err := tasks.Create(ctx, runtimedomain.NewPending("t-online", "s", "c", "inputs/t-online", now)); err != nil {
		t.Fatal(err)
	}
	bus := &captureBus{}
	n := &memNotify{}
	reg := static.New(instance.Instance{ID: "gpu-1", DispatchTopic: "dispatch.gpu-1"})
	svc := orchestrator.New(tasks, reg, bus, n)
	svc.Now = func() time.Time { return now }
	svc.Prep = stubPrep{}
	svc.Online = func(context.Context, sharedkernel.InstanceID) bool { return false }

	if err := svc.OnTaskCreated(ctx, sharedkernel.TaskCreated{TaskID: "t-online"}); err != nil {
		t.Fatal(err)
	}
	if len(bus.msgs) != 0 {
		t.Fatalf("expected no dispatch, got %d", len(bus.msgs))
	}
	task, err := tasks.Get(ctx, "t-online")
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != sharedkernel.TaskPending {
		t.Fatalf("status=%s", task.Status)
	}
}

func TestDispatchSetsJobRefWhenPrepSet(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(1, 0).UTC()
	if err := tasks.Create(ctx, runtimedomain.NewPending("t-prep", "s", "c", "inputs/t-prep", now)); err != nil {
		t.Fatal(err)
	}
	bus := &captureBus{}
	n := &memNotify{}
	reg := static.New(instance.Instance{ID: "gpu-1", DispatchTopic: "dispatch.gpu-1"})
	svc := orchestrator.New(tasks, reg, bus, n)
	svc.Now = func() time.Time { return now }
	svc.Prep = jobPrepStub{ref: sharedkernel.BlobRef{Key: "jobs/t-prep/job.json"}}

	if err := svc.OnTaskCreated(ctx, sharedkernel.TaskCreated{TaskID: "t-prep"}); err != nil {
		t.Fatal(err)
	}
	if len(bus.msgs) != 0 {
		t.Fatalf("must not publish, got %d", len(bus.msgs))
	}
	got, err := tasks.Get(ctx, "t-prep")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != sharedkernel.TaskQueued || got.JobRef.Key != "jobs/t-prep/job.json" {
		t.Fatalf("got %+v", got)
	}
}

type jobPrepStub struct {
	ref sharedkernel.BlobRef
}

func (j jobPrepStub) PrepareJob(context.Context, sharedkernel.TaskID, sharedkernel.InstanceID) (sharedkernel.BlobRef, error) {
	return j.ref, nil
}

func TestDispatchClaimableUsesEnabledWithoutOnline(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(1, 0).UTC()
	if err := tasks.Create(ctx, runtimedomain.NewPending("t-claim-enabled", "s", "c", "inputs/t", now)); err != nil {
		t.Fatal(err)
	}
	n := &memNotify{}
	reg := &enabledOnlyRegistry{items: []instance.Instance{{ID: "edge-1", DispatchTopic: "dispatch.edge-1"}}}
	svc := orchestrator.New(tasks, reg, nil, n)
	svc.Now = func() time.Time { return now }
	svc.Prep = stubPrep{}

	if err := svc.OnTaskCreated(ctx, sharedkernel.TaskCreated{TaskID: "t-claim-enabled"}); err != nil {
		t.Fatal(err)
	}
	got, err := tasks.Get(ctx, "t-claim-enabled")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != sharedkernel.TaskQueued || got.InstanceID != "edge-1" {
		t.Fatalf("claimable dispatch must use ListEnabled when Dispatch is nil, got %+v", got)
	}
}

func TestDispatchUsesOnlineEvenWhenInstanceMarkedUnhealthy(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(1, 0).UTC()
	if err := tasks.Create(ctx, runtimedomain.NewPending("t-edge-health", "s", "c", "inputs/t", now)); err != nil {
		t.Fatal(err)
	}
	bus := &captureBus{}
	n := &memNotify{}
	reg := &enabledOnlyRegistry{items: []instance.Instance{{ID: "edge-1", DispatchTopic: "dispatch.edge-1"}}}
	svc := orchestrator.New(tasks, reg, bus, n)
	svc.Now = func() time.Time { return now }
	svc.Prep = stubPrep{}
	svc.Online = func(context.Context, sharedkernel.InstanceID) bool { return true }

	if err := svc.OnTaskCreated(ctx, sharedkernel.TaskCreated{TaskID: "t-edge-health"}); err != nil {
		t.Fatal(err)
	}
	got, err := tasks.Get(ctx, "t-edge-health")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != sharedkernel.TaskQueued || got.InstanceID != "edge-1" {
		t.Fatalf("expected claimable on edge-1, got %+v", got)
	}
	if len(bus.msgs) != 0 {
		t.Fatalf("must not publish, got %d", len(bus.msgs))
	}
}

// enabledOnlyRegistry reports no healthy instances but ListEnabled returns items
// (simulates split: cloud Comfy probe fails, Edge heartbeat is online).
type enabledOnlyRegistry struct {
	items []instance.Instance
}

func (r *enabledOnlyRegistry) ListHealthy(context.Context, instance.CapabilityFilter) ([]instance.Instance, error) {
	return nil, nil
}

func (r *enabledOnlyRegistry) ListEnabled(_ context.Context, _ instance.CapabilityFilter) ([]instance.Instance, error) {
	out := make([]instance.Instance, len(r.items))
	copy(out, r.items)
	return out, nil
}

func (r *enabledOnlyRegistry) Get(_ context.Context, id sharedkernel.InstanceID) (*instance.Instance, error) {
	for i := range r.items {
		if r.items[i].ID == id {
			cp := r.items[i]
			return &cp, nil
		}
	}
	return nil, instance.ErrNotFound
}
