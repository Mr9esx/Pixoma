package domain

import (
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestBackoffFor(t *testing.T) {
	cases := []struct {
		attempt int
		want    time.Duration
	}{
		{1, 5 * time.Second},
		{2, 15 * time.Second},
		{3, 45 * time.Second},
		{9, 45 * time.Second},
		{0, 5 * time.Second},
	}
	for _, tc := range cases {
		if got := BackoffFor(tc.attempt); got != tc.want {
			t.Fatalf("BackoffFor(%d) = %v, want %v", tc.attempt, got, tc.want)
		}
	}
}

func TestRequeueAfterFailure_BoundedRetries(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	task := NewPending("t1", "s1", sharedkernel.CaseID(1), "in", now)
	_ = task.PrepareForTopic("fast-gpu", sharedkernel.BlobRef{Key: "j"}, now)
	_ = task.ClaimWithLease("gpu-1", time.Minute, now)

	// Failure 1 → requeued with 5s backoff, edge cleared, topic kept.
	requeued, err := task.RequeueAfterFailure(BackoffFor(1), now, "boom 1")
	if err != nil || !requeued {
		t.Fatalf("requeue1: requeued=%v err=%v", requeued, err)
	}
	if task.Status != sharedkernel.TaskQueued || task.EdgeID != "" || task.DispatchTopic != "fast-gpu" {
		t.Fatalf("after failure 1: %+v", task)
	}
	if task.RequeueAt.Sub(now) != 5*time.Second {
		t.Fatalf("requeue_at = %v", task.RequeueAt.Sub(now))
	}

	// Failures 2 and 3 requeue with 15s/45s.
	_ = task.ClaimWithLease("gpu-2", time.Minute, now)
	requeued, _ = task.RequeueAfterFailure(BackoffFor(2), now, "boom 2")
	if !requeued || task.RequeueAt.Sub(now) != 15*time.Second {
		t.Fatalf("failure 2: requeued=%v requeue_at=%v", requeued, task.RequeueAt.Sub(now))
	}
	_ = task.ClaimWithLease("gpu-2", time.Minute, now)
	requeued, _ = task.RequeueAfterFailure(BackoffFor(3), now, "boom 3")
	if !requeued || task.RequeueAt.Sub(now) != 45*time.Second {
		t.Fatalf("failure 3: requeued=%v requeue_at=%v", requeued, task.RequeueAt.Sub(now))
	}

	// Fourth failure exhausts retries → final failed.
	_ = task.ClaimWithLease("gpu-3", time.Minute, now)
	requeued, err = task.RequeueAfterFailure(BackoffFor(4), now, "boom 4")
	if err != nil {
		t.Fatalf("failure 4: %v", err)
	}
	if requeued {
		t.Fatal("expected final failure, got requeue")
	}
	if task.Status != sharedkernel.TaskFailed {
		t.Fatalf("status = %s, want failed", task.Status)
	}
	if task.Attempts != 4 {
		t.Fatalf("attempts = %d, want 4", task.Attempts)
	}
}

func TestRequeueAfterFailure_IdempotentOnFailed(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	task := NewPending("t1", "s1", sharedkernel.CaseID(1), "in", now)
	_ = task.MarkFailed("err", "boom", now)
	requeued, err := task.RequeueAfterFailure(time.Second, now, "again")
	if err != nil || requeued || task.Status != sharedkernel.TaskFailed {
		t.Fatalf("idempotent: requeued=%v err=%v status=%s", requeued, err, task.Status)
	}
}
