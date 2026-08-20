package domain

import (
	"context"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestMemory_LeaseExpiryRequeueKeepsAttempts(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryTaskRepository()
	now := time.Unix(10, 0).UTC()

	task := NewPending("t1", "s1", sharedkernel.CaseID(1), "in", now)
	_ = task.PrepareForTopic("fast-gpu", sharedkernel.BlobRef{Key: "j"}, now)
	_ = task.ClaimWithLease("gpu-1", time.Minute, now)
	task.Attempts = 2 // prior execution failures
	_ = repo.Create(ctx, task)

	later := now.Add(2 * time.Minute)
	n, err := repo.RequeueExpiredLeases(ctx, later)
	if err != nil || n != 1 {
		t.Fatalf("requeue n=%d err=%v", n, err)
	}
	got, _ := repo.Get(ctx, "t1")
	if got.Attempts != 2 {
		t.Fatalf("attempts changed to %d, want 2", got.Attempts)
	}
	if got.EdgeID != "" || got.Status != sharedkernel.TaskQueued {
		t.Fatalf("after requeue: status=%s edge=%q", got.Status, got.EdgeID)
	}
	if got.DispatchTopic != "fast-gpu" {
		t.Fatalf("topic = %q", got.DispatchTopic)
	}
	// Claimable immediately (requeue_at = now).
	claimed, err := repo.ClaimNextWithLease(ctx, "gpu-2", []string{"fast-gpu"}, time.Minute, later)
	if err != nil || claimed == nil || claimed.ID != "t1" {
		t.Fatalf("reclaim: %v %v", claimed, err)
	}
}
