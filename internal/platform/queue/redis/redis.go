package redis

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue"
)

const (
	fieldTopic   = "topic"
	fieldKey     = "key"
	fieldPayload = "payload"
)

// Options configures a Redis Streams queue bus.
type Options struct {
	Client        goredis.Cmdable
	ConsumerGroup string
	ConsumerName  string
	// StreamPrefix prepended to topic when forming stream keys (default "q:").
	StreamPrefix string
	Block        time.Duration
}

// Bus implements queue.Bus on Redis Streams.
type Bus struct {
	rdb      goredis.Cmdable
	group    string
	consumer string
	prefix   string
	block    time.Duration

	rootCtx context.Context
	cancel  context.CancelFunc

	mu     sync.Mutex
	closed bool
	wg     sync.WaitGroup
}

// New creates a Redis Streams bus. Client is required.
func New(opts Options) (*Bus, error) {
	if opts.Client == nil {
		return nil, errors.New("queue/redis: nil client")
	}
	group := strings.TrimSpace(opts.ConsumerGroup)
	if group == "" {
		group = "comfyui-tgbot"
	}
	name := strings.TrimSpace(opts.ConsumerName)
	if name == "" {
		name = "worker"
	}
	prefix := opts.StreamPrefix
	if prefix == "" {
		prefix = "q:"
	}
	block := opts.Block
	if block <= 0 {
		block = time.Second
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Bus{
		rdb:      opts.Client,
		group:    group,
		consumer: name,
		prefix:   prefix,
		block:    block,
		rootCtx:  ctx,
		cancel:   cancel,
	}, nil
}

func (b *Bus) streamKey(topic string) string {
	return b.prefix + topic
}

func (b *Bus) Publish(ctx context.Context, msg queue.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	b.mu.Lock()
	closed := b.closed
	b.mu.Unlock()
	if closed {
		return errors.New("queue/redis: bus closed")
	}
	if strings.TrimSpace(msg.Topic) == "" {
		return errors.New("queue/redis: empty topic")
	}
	values := map[string]any{
		fieldTopic:   msg.Topic,
		fieldKey:     msg.Key,
		fieldPayload: string(msg.Payload),
	}
	if err := b.rdb.XAdd(ctx, &goredis.XAddArgs{
		Stream: b.streamKey(msg.Topic),
		Values: values,
	}).Err(); err != nil {
		return fmt.Errorf("queue/redis: xadd: %w", err)
	}
	return nil
}

func (b *Bus) Subscribe(ctx context.Context, topic string, h queue.Handler) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if h == nil {
		return errors.New("queue/redis: nil handler")
	}
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return errors.New("queue/redis: bus closed")
	}
	b.mu.Unlock()

	stream := b.streamKey(topic)
	if err := b.ensureGroup(ctx, stream); err != nil {
		return err
	}

	loopCtx, loopCancel := context.WithCancel(ctx)
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		defer loopCancel()
		select {
		case <-b.rootCtx.Done():
			loopCancel()
		case <-loopCtx.Done():
		}
	}()
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		b.consumeLoop(loopCtx, stream, topic, h)
	}()
	return nil
}

func (b *Bus) ensureGroup(ctx context.Context, stream string) error {
	err := b.rdb.XGroupCreateMkStream(ctx, stream, b.group, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("queue/redis: xgroup create: %w", err)
	}
	return nil
}

func (b *Bus) consumeLoop(ctx context.Context, stream, topic string, h queue.Handler) {
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		// Reclaim pending (PEL) before new messages so handler failures are retried.
		if b.consumeOnce(ctx, stream, topic, h, "0", 0) {
			continue
		}
		b.consumeOnce(ctx, stream, topic, h, ">", b.block)
	}
}

// consumeOnce reads up to one message from the group. Returns true if a message was handled
// (success or failure). id is ">" for new entries or "0" for pending.
func (b *Bus) consumeOnce(ctx context.Context, stream, topic string, h queue.Handler, id string, block time.Duration) bool {
	streams, err := b.rdb.XReadGroup(ctx, &goredis.XReadGroupArgs{
		Group:    b.group,
		Consumer: b.consumer,
		Streams:  []string{stream, id},
		Count:    1,
		Block:    block,
	}).Result()
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return false
		}
		if errors.Is(err, goredis.Nil) {
			return false
		}
		select {
		case <-ctx.Done():
		case <-time.After(100 * time.Millisecond):
		}
		return false
	}
	handled := false
	for _, s := range streams {
		for _, m := range s.Messages {
			handled = true
			msg := queue.Message{Topic: topic}
			if v, ok := m.Values[fieldTopic].(string); ok && v != "" {
				msg.Topic = v
			}
			if v, ok := m.Values[fieldKey].(string); ok {
				msg.Key = v
			}
			if v, ok := m.Values[fieldPayload].(string); ok {
				msg.Payload = []byte(v)
			}
			if err := h(ctx, msg); err != nil {
				// Leave in PEL for retry; brief backoff avoids a tight spin.
				select {
				case <-ctx.Done():
				case <-time.After(50 * time.Millisecond):
				}
				continue
			}
			_ = b.rdb.XAck(ctx, stream, b.group, m.ID).Err()
		}
	}
	return handled
}

func (b *Bus) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	b.cancel()
	b.mu.Unlock()
	b.wg.Wait()
	return nil
}

var _ queue.Bus = (*Bus)(nil)
