package scheduling_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/apps/pixoma/internal/scheduling"
	catalogdomain "github.com/Mr9esx/Pixoma/internal/cases/domain"
	edge "github.com/Mr9esx/Pixoma/internal/edge/domain"
	"github.com/Mr9esx/Pixoma/internal/edge/infrastructure/static"
	"github.com/Mr9esx/Pixoma/internal/platform/notify"
	"github.com/Mr9esx/Pixoma/internal/platform/queue"
	"github.com/Mr9esx/Pixoma/internal/platform/queue/memory"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	"github.com/Mr9esx/Pixoma/internal/tasks/application/orchestrator"
	runtimedomain "github.com/Mr9esx/Pixoma/internal/tasks/domain"
	"github.com/Mr9esx/Pixoma/internal/tasks/domain/condition"
)

func TestSubscribeTaskCreatedMakesClaimable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	now := time.Unix(1000, 0).UTC()
	tasks := runtimedomain.NewMemoryTaskRepository()
	if err := tasks.Create(ctx, runtimedomain.NewPending("t-bus", "s", sharedkernel.CaseID(1), "in", now)); err != nil {
		t.Fatal(err)
	}
	reg := static.New(edge.Instance{ID: "local", SubscribeTopics: []string{"default"}})
	orch := orchestrator.New(tasks, reg, nil, notify.Nop{})
	orch.Now = func() time.Time { return now }
	orch.Prep = jobPrep{ref: sharedkernel.BlobRef{Key: "jobs/t-bus/job.json"}}
	orch.Cases = explicitDefaultCaseReader{}
	orch.Condition = condition.NewRegistry()
	bus := memory.New()
	if err := scheduling.SubscribeTaskCreated(ctx, bus, orch); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(sharedkernel.TaskCreated{TaskID: "t-bus", CreatedAt: now})
	if err := bus.Publish(ctx, queue.Message{Topic: sharedkernel.TopicTaskCreated, Key: "t-bus", Payload: payload}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		got, err := tasks.Get(ctx, "t-bus")
		if err == nil && got.Status == sharedkernel.TaskQueued && got.JobRef.Key == "jobs/t-bus/job.json" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	got, _ := tasks.Get(ctx, "t-bus")
	t.Fatalf("task not claimable: %+v", got)
}

type jobPrep struct {
	ref sharedkernel.BlobRef
}

func (j jobPrep) PrepareJob(context.Context, sharedkernel.TaskID) (sharedkernel.BlobRef, error) {
	return j.ref, nil
}

type explicitDefaultCaseReader struct{}

func (explicitDefaultCaseReader) GetCase(context.Context, sharedkernel.CaseID) (*catalogdomain.CaseDocument, error) {
	return &catalogdomain.CaseDocument{
		Routing: &catalogdomain.RoutingConfig{Rules: []catalogdomain.RoutingRule{
			{When: []byte(`{"always":true}`), Topic: "default"},
		}},
	}, nil
}
