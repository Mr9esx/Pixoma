package queue

import (
	"context"
	"sync"
)

// SubscriptionSet tracks topics already subscribed so Ensure is idempotent.
type SubscriptionSet struct {
	mu     sync.Mutex
	topics map[string]struct{}
}

// Ensure subscribes h to topic once. Repeat calls for the same topic are no-ops.
func (s *SubscriptionSet) Ensure(ctx context.Context, sub Subscriber, topic string, h Handler) error {
	if s == nil {
		return sub.Subscribe(ctx, topic, h)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.topics == nil {
		s.topics = make(map[string]struct{})
	}
	if _, ok := s.topics[topic]; ok {
		return nil
	}
	if err := sub.Subscribe(ctx, topic, h); err != nil {
		return err
	}
	s.topics[topic] = struct{}{}
	return nil
}
