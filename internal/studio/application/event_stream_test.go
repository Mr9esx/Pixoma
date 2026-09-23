package application

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestEventHubDoesNotRetainRunHistory(t *testing.T) {
	hub := NewEventHub()
	ctx := context.Background()
	for sequence := uint64(1); sequence <= 3; sequence++ {
		if err := hub.Publish(ctx, LiveEvent{RunID: "run-history", Sequence: sequence}); err != nil {
			t.Fatal(err)
		}
	}
	history, _, unsubscribe := hub.SubscribeAfter("run-history", 0)
	defer unsubscribe()
	if len(history) != 0 {
		t.Fatalf("in-memory replay = %d events, want none", len(history))
	}
}

func TestEventHubSlowSubscriberCannotBlockPublish(t *testing.T) {
	hub := NewEventHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, _, unsubscribe := hub.Subscribe("run-slow")
	defer unsubscribe()
	for sequence := uint64(1); sequence <= 1024; sequence++ {
		if err := hub.Publish(ctx, LiveEvent{RunID: "run-slow", Sequence: sequence}); err != nil {
			t.Fatal(err)
		}
	}
	finished := make(chan error, 1)
	go func() {
		finished <- hub.Publish(ctx, LiveEvent{RunID: "run-slow", Sequence: 1025})
	}()
	select {
	case err := <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(100 * time.Millisecond):
		cancel()
		<-finished
		t.Fatal("slow subscriber blocked event publication")
	}
}

func TestEventHubSubscribeAfterOnlyNotifiesNewCommits(t *testing.T) {
	hub := NewEventHub()
	ctx := context.Background()
	for _, event := range []LiveEvent{
		{RunID: "run-1", Sequence: 1, Type: "TEXT_MESSAGE_START", Payload: json.RawMessage(`{}`)},
		{RunID: "run-1", Type: "TEXT_MESSAGE_CONTENT", Payload: json.RawMessage(`{"delta":"旧"}`)},
		{RunID: "run-1", Sequence: 2, Type: "APPROVAL_REQUIRED", Payload: json.RawMessage(`{}`)},
	} {
		if err := hub.Publish(ctx, event); err != nil {
			t.Fatal(err)
		}
	}
	history, events, unsubscribe := hub.SubscribeAfter("run-1", 2)
	defer unsubscribe()
	if len(history) != 0 {
		t.Fatalf("history before resume = %#v", history)
	}
	if err := hub.Publish(ctx, LiveEvent{RunID: "run-1", Sequence: 3, Type: "TEXT_MESSAGE_CONTENT", Payload: json.RawMessage(`{"delta":"新"}`)}); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-events:
		if event.Sequence != 3 || event.Type != "" || len(event.Payload) != 0 {
			t.Fatalf("notification retained payload: %#v", event)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}
