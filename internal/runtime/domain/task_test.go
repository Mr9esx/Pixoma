package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestTaskHappyPath(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	task := domain.NewPending("t1", "s1", "c1", "inputs/t1", now)
	if task.SessionID != "s1" {
		t.Fatalf("session_id=%q", task.SessionID)
	}
	if err := task.MarkQueued("inst-a", now); err != nil {
		t.Fatal(err)
	}
	if err := task.MarkRunning("p1", now); err != nil {
		t.Fatal(err)
	}
	if err := task.MarkSucceeded([]domain.OutputRef{{Key: "image", Blob: sharedkernel.BlobRef{Key: "out.png"}}}, now); err != nil {
		t.Fatal(err)
	}
	if task.Status != sharedkernel.TaskSucceeded {
		t.Fatalf("status=%s", task.Status)
	}
}

func TestCancelRunningForbidden(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	task := domain.NewPending("t1", "s1", "c1", "inputs/t1", now)
	_ = task.MarkQueued("i", now)
	_ = task.MarkRunning("p", now)
	if err := task.MarkCancelled(now); !errors.Is(err, domain.ErrCancelNotAllowed) {
		t.Fatalf("got %v", err)
	}
}

func TestCancelPendingOK(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	task := domain.NewPending("t1", "s1", "c1", "inputs/t1", now)
	if err := task.MarkCancelled(now); err != nil {
		t.Fatal(err)
	}
	if task.Status != sharedkernel.TaskCancelled {
		t.Fatal(task.Status)
	}
}

func TestInvalidSucceededFromPending(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	task := domain.NewPending("t1", "s1", "c1", "inputs/t1", now)
	if err := task.MarkSucceeded(nil, now); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("got %v", err)
	}
}

func TestMarkSucceededIdempotent(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	task := domain.NewPending("t1", "s1", "c1", "inputs/t1", now)
	_ = task.MarkQueued("i", now)
	_ = task.MarkRunning("p", now)
	_ = task.MarkSucceeded(nil, now)
	if err := task.MarkSucceeded(nil, now); err != nil {
		t.Fatal(err)
	}
}

func TestPrepareForClaim_SetsJobRefAndQueued(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	task := domain.NewPending("t1", "s1", "c1", "inputs/t1", now)
	ref := sharedkernel.BlobRef{Key: "jobs/t1/job.json"}
	if err := task.PrepareForClaim("gpu-1", ref, now); err != nil {
		t.Fatal(err)
	}
	if task.Status != sharedkernel.TaskQueued {
		t.Fatalf("status=%s", task.Status)
	}
	if task.InstanceID != "gpu-1" || task.JobRef.Key != ref.Key {
		t.Fatalf("instance/job_ref: %+v %+v", task.InstanceID, task.JobRef)
	}
}

func TestClaimWithLease_FromQueued(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	task := domain.NewPending("t1", "s1", "c1", "inputs/t1", now)
	_ = task.PrepareForClaim("gpu-1", sharedkernel.BlobRef{Key: "jobs/t1/job.json"}, now)
	lease := 90 * time.Second
	if err := task.ClaimWithLease("gpu-1", lease, now); err != nil {
		t.Fatal(err)
	}
	if task.Status != sharedkernel.TaskRunning {
		t.Fatalf("status=%s", task.Status)
	}
	if task.LeaseUntil.Sub(now) != lease {
		t.Fatalf("lease=%v", task.LeaseUntil)
	}
}

func TestClaimWithLease_WrongInstance(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	task := domain.NewPending("t1", "s1", "c1", "inputs/t1", now)
	_ = task.PrepareForClaim("gpu-1", sharedkernel.BlobRef{Key: "j"}, now)
	if err := task.ClaimWithLease("gpu-2", time.Minute, now); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("got %v", err)
	}
}

func TestRequeueIfLeaseExpired(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	task := domain.NewPending("t1", "s1", "c1", "inputs/t1", now)
	_ = task.PrepareForClaim("gpu-1", sharedkernel.BlobRef{Key: "j"}, now)
	_ = task.ClaimWithLease("gpu-1", time.Second, now)
	later := now.Add(2 * time.Second)
	requeued, err := task.RequeueIfLeaseExpired(later)
	if err != nil {
		t.Fatal(err)
	}
	if !requeued || task.Status != sharedkernel.TaskQueued {
		t.Fatalf("requeued=%v status=%s", requeued, task.Status)
	}
	if !task.LeaseUntil.IsZero() {
		t.Fatal("lease should clear")
	}
}
