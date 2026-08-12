package redis_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue"
	queueredis "github.com/mr9esx/comfyui_tgbot/internal/platform/queue/redis"
)

func TestPublishSubscribeRoundTrip(t *testing.T) {
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	bus, err := queueredis.New(queueredis.Options{
		Client:       client,
		ConsumerGroup: "test-group",
		ConsumerName: "edge-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = bus.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	var got queue.Message
	var wg sync.WaitGroup
	wg.Add(1)
	if err := bus.Subscribe(ctx, "dispatch.local", func(ctx context.Context, msg queue.Message) error {
		got = msg
		wg.Done()
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	payload := []byte(`{"task_id":"t1"}`)
	if err := bus.Publish(ctx, queue.Message{
		Topic:   "dispatch.local",
		Key:     "t1",
		Payload: payload,
	}); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for message")
	}

	if got.Topic != "dispatch.local" {
		t.Fatalf("topic=%q", got.Topic)
	}
	if got.Key != "t1" {
		t.Fatalf("key=%q", got.Key)
	}
	if string(got.Payload) != string(payload) {
		t.Fatalf("payload=%q", got.Payload)
	}
}

func TestHandlerFailureIsRetriedFromPEL(t *testing.T) {
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	bus, err := queueredis.New(queueredis.Options{
		Client:        client,
		ConsumerGroup: "retry-group",
		ConsumerName:  "c1",
		Block:         50 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = bus.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	var mu sync.Mutex
	attempts := 0
	done := make(chan struct{})
	if err := bus.Subscribe(ctx, "dispatch.retry", func(ctx context.Context, msg queue.Message) error {
		mu.Lock()
		attempts++
		n := attempts
		mu.Unlock()
		if n == 1 {
			return errors.New("transient")
		}
		close(done)
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if err := bus.Publish(ctx, queue.Message{
		Topic: "dispatch.retry", Key: "t1", Payload: []byte(`{}`),
	}); err != nil {
		t.Fatal(err)
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for PEL retry")
	}
	mu.Lock()
	defer mu.Unlock()
	if attempts < 2 {
		t.Fatalf("attempts=%d want >=2", attempts)
	}
}
