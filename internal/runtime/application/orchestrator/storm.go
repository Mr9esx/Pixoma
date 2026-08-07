package orchestrator

import (
	"sync"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type StormConfig struct {
	SchedulePerTick   int
	ReconcilePerTick  int
	RedispatchPerMin  int
	BreakerFailures   int
	BreakerCoolDown   time.Duration
}

func (c StormConfig) withDefaults() StormConfig {
	if c.SchedulePerTick <= 0 {
		c.SchedulePerTick = 32
	}
	if c.ReconcilePerTick <= 0 {
		c.ReconcilePerTick = 16
	}
	if c.RedispatchPerMin <= 0 {
		c.RedispatchPerMin = 60
	}
	if c.BreakerFailures <= 0 {
		c.BreakerFailures = 5
	}
	if c.BreakerCoolDown <= 0 {
		c.BreakerCoolDown = 30 * time.Second
	}
	return c
}

type TokenBucket struct {
	mu     sync.Mutex
	tokens int
	max    int
}

func NewTokenBucket(max int) *TokenBucket {
	return &TokenBucket{tokens: max, max: max}
}

func (b *TokenBucket) Allow(n int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.tokens < n {
		return false
	}
	b.tokens -= n
	return true
}

func (b *TokenBucket) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.tokens = b.max
}

type CircuitBreaker struct {
	mu        sync.Mutex
	failures  map[sharedkernel.InstanceID]int
	openUntil map[sharedkernel.InstanceID]time.Time
	threshold int
	coolDown  time.Duration
	now       func() time.Time
}

func NewCircuitBreaker(threshold int, coolDown time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failures:  map[sharedkernel.InstanceID]int{},
		openUntil: map[sharedkernel.InstanceID]time.Time{},
		threshold: threshold,
		coolDown:  coolDown,
		now:       time.Now,
	}
}

func (c *CircuitBreaker) Allow(id sharedkernel.InstanceID) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if until, ok := c.openUntil[id]; ok && c.now().Before(until) {
		return false
	}
	return true
}

func (c *CircuitBreaker) RecordFailure(id sharedkernel.InstanceID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures[id]++
	if c.failures[id] >= c.threshold {
		c.openUntil[id] = c.now().Add(c.coolDown)
		c.failures[id] = 0
	}
}

func (c *CircuitBreaker) RecordSuccess(id sharedkernel.InstanceID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures[id] = 0
	delete(c.openUntil, id)
}

type StormGuard struct {
	cfg       StormConfig
	schedule  *TokenBucket
	reconcile *TokenBucket
	Breaker   *CircuitBreaker
}

func NewStormGuard(cfg StormConfig) *StormGuard {
	cfg = cfg.withDefaults()
	return &StormGuard{
		cfg:       cfg,
		schedule:  NewTokenBucket(cfg.SchedulePerTick),
		reconcile: NewTokenBucket(cfg.ReconcilePerTick),
		Breaker:   NewCircuitBreaker(cfg.BreakerFailures, cfg.BreakerCoolDown),
	}
}

func (g *StormGuard) AllowSchedule(n int) bool  { return g.schedule.Allow(n) }
func (g *StormGuard) AllowReconcile(n int) bool { return g.reconcile.Allow(n) }

func (g *StormGuard) ResetTick() {
	g.schedule.Reset()
	g.reconcile.Reset()
}
