package setup

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// AttemptLimiter is a small fixed-window limiter for authentication endpoints.
// It is intentionally in-process: deployments behind multiple replicas should
// replace it with a shared store before exposing login publicly.
type AttemptLimiter struct {
	mu       sync.Mutex
	max      int
	window   time.Duration
	attempts map[string]attempt
	Now      func() time.Time
}

type attempt struct {
	count     int
	startedAt time.Time
}

func NewAttemptLimiter(max int, window time.Duration) *AttemptLimiter {
	if max <= 0 {
		max = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	return &AttemptLimiter{
		max:      max,
		window:   window,
		attempts: make(map[string]attempt),
		Now:      time.Now,
	}
}

func (l *AttemptLimiter) Allowed(key string) bool {
	if l == nil {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	current, exists := l.attempts[key]
	if exists && now.Sub(current.startedAt) >= l.window {
		delete(l.attempts, key)
		exists = false
	}
	return !exists || current.count < l.max
}

func (l *AttemptLimiter) Record(key string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	current, exists := l.attempts[key]
	if !exists || now.Sub(current.startedAt) >= l.window {
		current = attempt{startedAt: now}
	}
	current.count++
	l.attempts[key] = current
}

func (l *AttemptLimiter) Reset(key string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, key)
}

func (l *AttemptLimiter) now() time.Time {
	if l.Now != nil {
		return l.Now()
	}
	return time.Now()
}

func clientKey(r *http.Request, username string) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if username == "" {
		return host
	}
	return username + "|" + host
}
