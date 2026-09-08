package setup_test

import (
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/httpapi/setup"
)

func TestAttemptLimiterBlocksAndResets(t *testing.T) {
	limiter := setup.NewAttemptLimiter(2, time.Minute)
	if !limiter.Allowed("key") {
		t.Fatal("first two attempts must be allowed")
	}
	limiter.Record("key")
	if !limiter.Allowed("key") {
		t.Fatal("second attempt must be allowed")
	}
	limiter.Record("key")
	if limiter.Allowed("key") {
		t.Fatal("third attempt must be blocked")
	}
	limiter.Reset("key")
	if !limiter.Allowed("key") {
		t.Fatal("reset must allow a new attempt")
	}
}

func TestAttemptLimiterExpiry(t *testing.T) {
	now := time.Now()
	limiter := setup.NewAttemptLimiter(1, time.Minute)
	limiter.Now = func() time.Time { return now }
	if !limiter.Allowed("key") {
		t.Fatal("first attempt must be allowed")
	}
	limiter.Now = func() time.Time { return now.Add(61 * time.Second) }
	if !limiter.Allowed("key") {
		t.Fatal("expired attempt must be allowed")
	}
}
