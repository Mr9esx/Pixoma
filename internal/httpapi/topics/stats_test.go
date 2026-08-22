package topics

import (
	"testing"
	"time"

	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestAggregateTopicStats(t *testing.T) {
	from := time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	started := from.Add(30 * time.Minute)

	mk := func(id string, status sharedkernel.TaskStatus, code string, startedAt, completedAt time.Time) *runtimedomain.Task {
		return &runtimedomain.Task{
			ID:            sharedkernel.TaskID(id),
			Status:        status,
			DispatchTopic: "fast-gpu",
			ErrorCode:     code,
			StartedAt:     startedAt,
			CompletedAt:   completedAt,
			CreatedAt:     startedAt,
			UpdatedAt:     completedAt,
		}
	}

	tasks := []*runtimedomain.Task{
		mk("a", sharedkernel.TaskSucceeded, "", started, started.Add(10*time.Second)),
		mk("b", sharedkernel.TaskFailed, "timeout", started, started.Add(5*time.Second)),
		mk("c", sharedkernel.TaskSucceeded, "", started.Add(time.Hour), started.Add(time.Hour+20*time.Second)),
	}

	got := aggregateTopicStats(tasks, from, to)
	if got.TaskCount != 3 {
		t.Fatalf("task_count = %d, want 3", got.TaskCount)
	}
	if got.Status[string(sharedkernel.TaskSucceeded)] != 2 {
		t.Fatalf("succeeded = %d, want 2", got.Status[string(sharedkernel.TaskSucceeded)])
	}
	if got.Status[string(sharedkernel.TaskFailed)] != 1 {
		t.Fatalf("failed = %d, want 1", got.Status[string(sharedkernel.TaskFailed)])
	}
	if got.SuccessRate == nil || *got.SuccessRate < 0.66 || *got.SuccessRate > 0.67 {
		t.Fatalf("success_rate = %v, want ~0.667", got.SuccessRate)
	}
	if len(got.ErrorCodes) != 1 || got.ErrorCodes[0].Code != "timeout" {
		t.Fatalf("error_codes = %+v, want [{timeout 1}]", got.ErrorCodes)
	}
	if got.RuntimeMS.Count != 3 || got.RuntimeMS.SumMS != 35000 {
		t.Fatalf("runtime = %+v, want count=3 sum=35000", got.RuntimeMS)
	}
	if got.RuntimeMS.AvgMS == nil || *got.RuntimeMS.AvgMS < 11666 || *got.RuntimeMS.AvgMS > 11667 {
		t.Fatalf("avg_ms = %v, want ~11666", got.RuntimeMS.AvgMS)
	}
	// 24h 窗口 → 小时桶：25 个桶，前 2 个桶有任务。
	if len(got.Throughput) != 25 {
		t.Fatalf("throughput buckets = %d, want 25", len(got.Throughput))
	}
	if got.Throughput[0].Count != 2 || got.Throughput[1].Count != 1 {
		t.Fatalf("throughput[0]=%d throughput[1]=%d, want 2/1", got.Throughput[0].Count, got.Throughput[1].Count)
	}
}

func TestTaskRuntimeMS_FallsBackToWallClock(t *testing.T) {
	created := time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC)
	task := &runtimedomain.Task{
		Status:    sharedkernel.TaskSucceeded,
		CreatedAt: created,
		UpdatedAt: created.Add(3 * time.Second),
	}
	ms, ok := taskRuntimeMS(task)
	if !ok || ms != 3000 {
		t.Fatalf("runtime = (%d, %v), want (3000, true)", ms, ok)
	}
}
