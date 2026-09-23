package application

import (
	"context"
	"encoding/json"
	"sync"
)

// LiveEvent notifies active AG-UI connections that a committed event is ready
// to be read from the durable Run ledger.
type LiveEvent struct {
	RunID    string
	Sequence uint64
	Type     string
	Payload  json.RawMessage
}

// EventStream is a best-effort notification channel, not a replay source.
type EventStream interface {
	Publish(context.Context, LiveEvent) error
	Subscribe(runID string) ([]LiveEvent, <-chan LiveEvent, func())
}

// EventStreamAfter keeps the older subscription API for protocol adapters;
// durable replay always happens through the repository.
type EventStreamAfter interface {
	SubscribeAfter(runID string, after uint64) ([]LiveEvent, <-chan LiveEvent, func())
}

// EventHub only tracks current subscribers. A slow subscriber cannot delay
// the Agent because AG-UI can always re-read committed events from storage.
type EventHub struct {
	mu   sync.Mutex
	runs map[string]*eventRun
}

type eventRun struct {
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
	if err := ctx.Err(); err != nil {
		return err
	}
	h.mu.Lock()
	run := h.runs[event.RunID]
	if run == nil {
		h.mu.Unlock()
		return nil
	}
	notice := LiveEvent{RunID: event.RunID, Sequence: event.Sequence}
	for subscriber := range run.subscribers {
		select {
		case <-subscriber.done:
			continue
		default:
		}
		select {
		case subscriber.events <- notice:
		default:
		}
	}
	h.mu.Unlock()
	return nil
}

func (h *EventHub) Subscribe(runID string) ([]LiveEvent, <-chan LiveEvent, func()) {
	return h.SubscribeAfter(runID, 0)
}

func (h *EventHub) SubscribeAfter(runID string, _ uint64) ([]LiveEvent, <-chan LiveEvent, func()) {
	subscriber := &eventSubscription{
		events: make(chan LiveEvent, 1),
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
	run.subscribers[subscriber] = struct{}{}
	h.mu.Unlock()

	unsubscribe := func() {
		subscriber.once.Do(func() {
			close(subscriber.done)
			h.mu.Lock()
			if current := h.runs[runID]; current != nil {
				delete(current.subscribers, subscriber)
				if len(current.subscribers) == 0 {
					delete(h.runs, runID)
				}
			}
			h.mu.Unlock()
		})
	}
	return []LiveEvent{}, subscriber.events, unsubscribe
}

var _ EventStream = (*EventHub)(nil)
