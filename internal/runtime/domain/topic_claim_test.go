package domain

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestMemory_ClaimFiltersByTopicAndRequeueAt(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryTaskRepository()
	now := time.Unix(10, 0).UTC()

	for _, tc := range []struct {
		id    string
		topic string
		wait  time.Time
		empty bool
		age   int
	}{
		{"fast-1", "fast-gpu", time.Time{}, false, 0},
		{"fast-2", "fast-gpu", now.Add(time.Minute), false, 1}, // requeue_at in future → not claimable
		{"other-1", "other", time.Time{}, false, 2},
		{"legacy-1", "default", time.Time{}, true, 3}, // empty dispatch_topic = default fallback
	} {
		task := NewPending(sharedkernel.TaskID(tc.id), "s", sharedkernel.CaseID(1), "in", now)
		task.CreatedAt = now.Add(time.Duration(tc.age) * time.Second)
		_ = task.PrepareForTopic(tc.topic, sharedkernel.BlobRef{Key: "j/" + tc.id}, now)
		if tc.empty {
			task.DispatchTopic = ""
		}
		task.RequeueAt = tc.wait
		_ = repo.Create(ctx, task)
	}

	claimed, err := repo.ClaimNextWithLease(ctx, "gpu-1", []string{"fast-gpu", "default"}, time.Minute, now)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if claimed == nil || claimed.ID != "fast-1" {
		t.Fatalf("claimed = %v, want fast-1 (requeue_at blocks fast-2, topic filters other-1)", claimed)
	}
	// fast-2 requeued in future stays; legacy-1 is claimable next
	second, err := repo.ClaimNextWithLease(ctx, "gpu-1", []string{"fast-gpu", "default"}, time.Minute, now)
	if err != nil || second == nil || second.ID != "legacy-1" {
		t.Fatalf("second = %v err=%v, want legacy-1", second, err)
	}
	// After requeue_at passes, fast-2 becomes claimable.
	later := now.Add(2 * time.Minute)
	third, err := repo.ClaimNextWithLease(ctx, "gpu-1", []string{"fast-gpu", "default"}, time.Minute, later)
	if err != nil || third == nil || third.ID != "fast-2" {
		t.Fatalf("third = %v err=%v, want fast-2", third, err)
	}
}

func TestMemory_ConcurrentClaimEachTaskOnce(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryTaskRepository()
	now := time.Unix(10, 0).UTC()

	const nTasks = 20
	for i := 0; i < nTasks; i++ {
		id := sharedkernel.TaskID(string(rune('a'+i)))
		task := NewPending(id, "s", sharedkernel.CaseID(1), "in", now)
		_ = task.PrepareForTopic("fast-gpu", sharedkernel.BlobRef{Key: "j/" + string(id)}, now)
		_ = repo.Create(ctx, task)
	}

	const workers = 8
	results := make(chan sharedkernel.TaskID, nTasks)
	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for {
				claimed, err := repo.ClaimNextWithLease(ctx, sharedkernel.EdgeID("gpu"), []string{"fast-gpu"}, time.Minute, now)
				if err != nil {
					t.Errorf("claim: %v", err)
					return
				}
				if claimed == nil {
					return
				}
				results <- claimed.ID
			}
		}()
	}
	wg.Wait()
	close(results)

	seen := map[sharedkernel.TaskID]int{}
	for id := range results {
		seen[id]++
	}
	if len(seen) != nTasks {
		t.Fatalf("claimed %d distinct tasks, want %d", len(seen), nTasks)
	}
	for id, n := range seen {
		if n != 1 {
			t.Fatalf("task %s claimed %d times", id, n)
		}
	}
}
