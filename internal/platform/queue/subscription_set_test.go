package queue_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/Mr9esx/Pixoma/internal/platform/queue"
	"github.com/Mr9esx/Pixoma/internal/platform/queue/memory"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

func TestSubscriptionSet_EnsureIdempotentAndDeliversNewTopic(t *testing.T) {
	bus := memory.New()
	defer bus.Close()
	ctx := context.Background()

	var hits atomic.Int32
	h := func(_ context.Context, _ queue.Message) error {
		hits.Add(1)
		return nil
	}

	var set queue.SubscriptionSet
	topicA := sharedkernel.TopicDispatch("gpu-a")
	topicB := sharedkernel.TopicDispatch("gpu-b")

	if err := set.Ensure(ctx, bus, topicA, h); err != nil {
		t.Fatalf("ensure a: %v", err)
	}
	if err := set.Ensure(ctx, bus, topicA, h); err != nil {
		t.Fatalf("ensure a again: %v", err)
	}
	if err := bus.Publish(ctx, queue.Message{Topic: topicA, Payload: []byte(`{}`)}); err != nil {
		t.Fatal(err)
	}
	if got := hits.Load(); got != 1 {
		t.Fatalf("hits after a=%d want 1 (idempotent subscribe)", got)
	}

	if err := set.Ensure(ctx, bus, topicB, h); err != nil {
		t.Fatalf("ensure b: %v", err)
	}
	if err := bus.Publish(ctx, queue.Message{Topic: topicB, Payload: []byte(`{}`)}); err != nil {
		t.Fatal(err)
	}
	if got := hits.Load(); got != 2 {
		t.Fatalf("hits after b=%d want 2", got)
	}
}
