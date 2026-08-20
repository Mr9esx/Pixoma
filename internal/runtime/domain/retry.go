package domain

import (
	"strconv"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// MaxRetries bounds the number of execution retries after task failures.
const MaxRetries = 3

// BackoffSchedule is the per-retry delay in order.
var BackoffSchedule = []time.Duration{5 * time.Second, 15 * time.Second, 45 * time.Second}

// BackoffFor returns the backoff delay for the given retry attempt (1-based).
func BackoffFor(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	idx := attempt - 1
	if idx >= len(BackoffSchedule) {
		idx = len(BackoffSchedule) - 1
	}
	return BackoffSchedule[idx]
}

// RequeueAfterFailure applies bounded retry semantics: the task returns to the
// queued state on its dispatch_topic with a backoff, or converges to final
// failed once retries are exhausted. Returns true when requeued.
func (t *Task) RequeueAfterFailure(backoff time.Duration, now time.Time, reason string) (bool, error) {
	if t.Status == sharedkernel.TaskFailed {
		return false, nil
	}
	t.Attempts++
	if t.Attempts > MaxRetries {
		t.Status = sharedkernel.TaskFailed
		t.ErrorCode = "max_retries"
		t.ErrorMessage = "exhausted retries (" + strconv.Itoa(MaxRetries) + "): " + reason
		t.EdgeID = ""
		t.LeaseUntil = time.Time{}
		t.RequeueAt = time.Time{}
		t.UpdatedAt = now
		return false, nil
	}
	t.Status = sharedkernel.TaskQueued
	t.EdgeID = ""
	t.LeaseUntil = time.Time{}
	t.RequeueAt = now.Add(backoff)
	t.ErrorCode = ""
	t.ErrorMessage = "retry " + strconv.Itoa(t.Attempts) + ": " + reason
	t.UpdatedAt = now
	return true, nil
}
