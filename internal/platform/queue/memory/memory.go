package memory

import (
	"context"
	"errors"
	"sync"

	"github.com/Mr9esx/Pixoma/internal/platform/queue"
)

type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]queue.Handler
	closed   bool
}

func New() *Bus {
	return &Bus{handlers: make(map[string][]queue.Handler)}
}

func (b *Bus) Publish(ctx context.Context, msg queue.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return errors.New("queue: bus closed")
	}
	handlers := append([]queue.Handler(nil), b.handlers[msg.Topic]...)
	b.mu.RUnlock()

	var firstErr error
	for _, h := range handlers {
		if err := h(ctx, msg); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (b *Bus) Subscribe(ctx context.Context, topic string, h queue.Handler) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if h == nil {
		return errors.New("queue: nil handler")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return errors.New("queue: bus closed")
	}
	b.handlers[topic] = append(b.handlers[topic], h)
	return nil
}

func (b *Bus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	b.handlers = nil
	return nil
}

var _ queue.Bus = (*Bus)(nil)
