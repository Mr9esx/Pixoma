package orchestrator_test

import (
	"context"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge/static"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/taskstats"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/application/orchestrator"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type fakeStatsRepo struct {
	calls []taskstats.AddTerminalInput
}

func (f *fakeStatsRepo) AddTerminal(_ context.Context, in taskstats.AddTerminalInput) error {
	f.calls = append(f.calls, in)
	return nil
}
func (f *fakeStatsRepo) ListDaily(context.Context, string, string) ([]taskstats.DailyRow, error) {
	return nil, nil
}
func (f *fakeStatsRepo) ListErrors(context.Context, string, string, int) ([]taskstats.ErrorRow, error) {
	return nil, nil
}
func (f *fakeStatsRepo) ListEdges(context.Context, string, string) ([]taskstats.EdgeRow, error) {
	return nil, nil
}
func (f *fakeStatsRepo) ListCases(context.Context, string, string, int) ([]taskstats.CaseRow, error) {
	return nil, nil
}
func (f *fakeStatsRepo) Prune(context.Context, string) error { return nil }

func TestApplyStatusRecordsTerminalStats(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(50, 0).UTC()
	task := runtimedomain.NewPending("t1", "s1", sharedkernel.CaseID(1), "inputs/t1", now)
	task.ChatID = "tg:9"
	_ = task.MarkQueued("local", now)
	_ = task.MarkRunning("p", now)
	if err := tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}

	stats := &fakeStatsRepo{}
	svc := orchestrator.New(tasks, static.New(edge.Instance{ID: "local"}), &captureBus{}, &memNotify{})
	svc.Stats = stats
	svc.Now = func() time.Time { return now }

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
	if len(stats.calls) != 1 {
		t.Fatalf("stats calls=%d, want 1 (duplicate terminal event must not re-record)", len(stats.calls))
	}
	got := stats.calls[0]
	if got.Status != taskstats.StatusSucceeded || got.EdgeID != "local" || !got.CompletedAt.Equal(now) {
		t.Fatalf("stats input: %+v", got)
	}
	if got.CaseID != 1 {
		t.Fatalf("stats case_id: %+v", got)
	}
}

func TestApplyStatusRecordsFailedWhenRetriesExhausted(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(60, 0).UTC()
	task := runtimedomain.NewPending("t2", "s2", sharedkernel.CaseID(1), "inputs/t2", now)
	_ = task.MarkQueued("local", now)
	_ = task.MarkRunning("p", now)
	task.Attempts = runtimedomain.MaxRetries
	if err := tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}

	stats := &fakeStatsRepo{}
	svc := orchestrator.New(tasks, static.New(edge.Instance{ID: "local"}), &captureBus{}, &memNotify{})
	svc.Stats = stats
	svc.Now = func() time.Time { return now }

	ev := sharedkernel.TaskStatusEvent{TaskID: "t2", Status: sharedkernel.TaskFailed, ErrorMsg: "boom", At: now}
	if err := svc.OnStatus(ctx, ev); err != nil {
		t.Fatal(err)
	}
	if len(stats.calls) != 1 {
		t.Fatalf("stats calls=%d, want 1", len(stats.calls))
	}
	got := stats.calls[0]
	if got.Status != taskstats.StatusFailed || got.ErrorCode != "max_retries" || got.CompletedAt.IsZero() {
		t.Fatalf("stats input: %+v", got)
	}
}

func TestRequestCancelRecordsTerminalStats(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(70, 0).UTC()
	task := runtimedomain.NewPending("t3", "s3", sharedkernel.CaseID(1), "inputs/t3", now)
	if err := tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}

	stats := &fakeStatsRepo{}
	svc := orchestrator.New(tasks, static.New(edge.Instance{ID: "local"}), &captureBus{}, &memNotify{})
	svc.Stats = stats
	svc.Now = func() time.Time { return now }

	if err := svc.RequestCancel(ctx, "t3"); err != nil {
		t.Fatal(err)
	}
	if len(stats.calls) != 1 {
		t.Fatalf("stats calls=%d, want 1", len(stats.calls))
	}
	got := stats.calls[0]
	if got.Status != taskstats.StatusCancelled || !got.CompletedAt.Equal(now) {
		t.Fatalf("stats input: %+v", got)
	}
}

func TestApplyStatusRunningDoesNotRecordStats(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(80, 0).UTC()
	task := runtimedomain.NewPending("t4", "s4", sharedkernel.CaseID(1), "inputs/t4", now)
	_ = task.MarkQueued("local", now)
	if err := tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}

	stats := &fakeStatsRepo{}
	svc := orchestrator.New(tasks, static.New(edge.Instance{ID: "local"}), &captureBus{}, &memNotify{})
	svc.Stats = stats
	svc.Now = func() time.Time { return now }

	ev := sharedkernel.TaskStatusEvent{TaskID: "t4", Status: sharedkernel.TaskRunning, PromptID: "p", At: now}
	if err := svc.OnStatus(ctx, ev); err != nil {
		t.Fatal(err)
	}
	if len(stats.calls) != 0 {
		t.Fatalf("stats calls=%d, want 0 for non-terminal event", len(stats.calls))
	}
}
