package memory_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue/memory"
)

func TestPublishSubscribeDeliversPayload(t *testing.T) {
	bus := memory.New()
	defer bus.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var (
		mu   sync.Mutex
		got  queue.Message
		done = make(chan struct{})
	)

	err := bus.Subscribe(ctx, "task.created", func(_ context.Context, msg queue.Message) error {
		mu.Lock()
		got = msg
		mu.Unlock()
		close(done)
		return nil
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	want := queue.Message{Topic: "task.created", Key: "t1", Payload: []byte(`{"ok":true}`)}
	if err := bus.Publish(ctx, want); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("timed out waiting for message")
	}

	mu.Lock()
	defer mu.Unlock()
	if got.Topic != want.Topic || got.Key != want.Key || string(got.Payload) != string(want.Payload) {
		t.Fatalf("got %+v want %+v", got, want)
	}
}
