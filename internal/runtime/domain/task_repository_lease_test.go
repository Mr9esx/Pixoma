package domain_test

import (
	"context"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestMemory_PrepareAndClaimWithLease(t *testing.T) {
	ctx := context.Background()
	repo := domain.NewMemoryTaskRepository()
	now := time.Unix(10, 0).UTC()
	if err := repo.Create(ctx, domain.NewPending("t1", "s1", "c1", "in", now)); err != nil {
		t.Fatal(err)
	}
	ref := sharedkernel.BlobRef{Key: "jobs/t1/job.json"}
	ok, err := repo.PrepareForClaim(ctx, "t1", "gpu-1", ref, now)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	claimed, err := repo.ClaimNextWithLease(ctx, "gpu-1", time.Minute, now)
	if err != nil || claimed == nil {
		t.Fatalf("claimed=%v err=%v", claimed, err)
	}
	if claimed.Status != sharedkernel.TaskRunning || claimed.JobRef.Key != ref.Key {
		t.Fatalf("%+v", claimed)
	}
	none, err := repo.ClaimNextWithLease(ctx, "gpu-1", time.Minute, now)
	if err != nil || none != nil {
		t.Fatalf("none=%v err=%v", none, err)
	}
}

func TestMemory_RequeueExpiredThenClaim(t *testing.T) {
	ctx := context.Background()
	repo := domain.NewMemoryTaskRepository()
	now := time.Unix(10, 0).UTC()
	task := domain.NewPending("t1", "s1", "c1", "in", now)
	_ = task.PrepareForClaim("gpu-1", sharedkernel.BlobRef{Key: "j"}, now)
	_ = task.ClaimWithLease("gpu-1", time.Second, now)
	if err := repo.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	later := now.Add(2 * time.Second)
	n, err := repo.RequeueExpiredLeases(ctx, later)
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	claimed, err := repo.ClaimNextWithLease(ctx, "gpu-1", time.Minute, later)
	if err != nil || claimed == nil || claimed.ID != "t1" {
		t.Fatalf("claimed=%v err=%v", claimed, err)
	}
}
