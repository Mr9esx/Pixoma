package domain

import (
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestPrepareForTopic(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	task := NewPending("t1", "s1", sharedkernel.CaseID(1), "inputs/t1", now)

	if err := task.PrepareForTopic("fast-gpu", sharedkernel.BlobRef{Key: "jobs/t1/job.json"}, now); err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if task.Status != sharedkernel.TaskQueued {
		t.Fatalf("status = %s, want queued", task.Status)
	}
	if task.DispatchTopic != "fast-gpu" {
		t.Fatalf("dispatch_topic = %q, want fast-gpu", task.DispatchTopic)
	}
	if task.EdgeID != "" {
		t.Fatalf("edge_id must stay empty after topic prepare, got %q", task.EdgeID)
	}
	if task.JobRef.Key == "" {
		t.Fatal("job_ref missing")
	}

	// second prepare must be rejected
	if err := task.PrepareForTopic("default", sharedkernel.BlobRef{Key: "other"}, now); err == nil {
		t.Fatal("expected ErrInvalidTransition on second prepare")
	}
}

func TestPrepareForTopicRequiresJobRef(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	task := NewPending("t2", "s2", sharedkernel.CaseID(1), "inputs/t2", now)
	if err := task.PrepareForTopic("default", sharedkernel.BlobRef{}, now); err == nil {
		t.Fatal("expected error for empty job_ref")
	}
}

func TestClaimWithLease_FromTopicPrepared(t *testing.T) {
	now := time.Unix(200, 0).UTC()
	task := NewPending("t3", "s3", sharedkernel.CaseID(1), "inputs/t3", now)
	_ = task.PrepareForTopic("default", sharedkernel.BlobRef{Key: "jobs/t3/job.json"}, now)

	if err := task.ClaimWithLease("gpu-1", 90*time.Second, now); err != nil {
		t.Fatalf("claim by any subscribed node: %v", err)
	}
	if task.Status != sharedkernel.TaskRunning || task.EdgeID != "gpu-1" {
		t.Fatalf("claim result: status=%s edge=%q", task.Status, task.EdgeID)
	}
	if task.LeaseUntil.Sub(now) != 90*time.Second {
		t.Fatalf("lease = %v", task.LeaseUntil.Sub(now))
	}
}
