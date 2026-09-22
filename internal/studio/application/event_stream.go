package application

import (
	"context"
	"encoding/json"
	"sync"
)

// LiveEvent is a runtime event that powers active AG-UI connections. Durable
// events keep their sequence; streamed deltas use sequence zero because they
// are intentionally not part of the run ledger.
type LiveEvent struct {
	RunID    string
	Sequence uint64
	Type     string
	Payload  json.RawMessage
}

// EventStream separates the high-frequency runtime stream from the durable
// Studio event ledger.
type EventStream interface {
	Publish(context.Context, LiveEvent) error
	Subscribe(runID string) ([]LiveEvent, <-chan LiveEvent, func())
}

// EventHub retains an active run's events for late AG-UI subscribers while
// broadcasting new events in their original order.
type EventHub struct {
	mu   sync.Mutex
	runs map[string]*eventRun
}

type eventRun struct {
	history     []LiveEvent
	subscribers map[*eventSubscription]struct{}
}

type eventSubscription struct {
	events chan LiveEvent
	done   chan struct{}
	once   sync.Once
}

func NewEventHub() *EventHub {
	return &EventHub{runs: map[string]*eventRun{}}
}

func (h *EventHub) Publish(ctx context.Context, event LiveEvent) error {
	if h == nil || event.RunID == "" {
		return nil
	}
	h.mu.Lock()
	run := h.runs[event.RunID]
	if run == nil {
		run = &eventRun{subscribers: map[*eventSubscription]struct{}{}}
		h.runs[event.RunID] = run
	}
	run.history = append(run.history, event)
	subscribers := make([]*eventSubscription, 0, len(run.subscribers))
	for subscriber := range run.subscribers {
		subscribers = append(subscribers, subscriber)
	}
	h.mu.Unlock()

	for _, subscriber := range subscribers {
		select {
		case subscriber.events <- event:
		case <-subscriber.done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (h *EventHub) Subscribe(runID string) ([]LiveEvent, <-chan LiveEvent, func()) {
	subscriber := &eventSubscription{
		events: make(chan LiveEvent, 1024),
		done:   make(chan struct{}),
	}
	if h == nil || runID == "" {
		return []LiveEvent{}, subscriber.events, func() {}
	}
	h.mu.Lock()
	run := h.runs[runID]
	if run == nil {
		run = &eventRun{subscribers: map[*eventSubscription]struct{}{}}
		h.runs[runID] = run
	}
	history := append([]LiveEvent{}, run.history...)
	run.subscribers[subscriber] = struct{}{}
	h.mu.Unlock()

	unsubscribe := func() {
		subscriber.once.Do(func() {
			close(subscriber.done)
			h.mu.Lock()
			if current := h.runs[runID]; current != nil {
				delete(current.subscribers, subscriber)
			}
			h.mu.Unlock()
		})
	}
	return history, subscriber.events, unsubscribe
}

var _ EventStream = (*EventHub)(nil)
