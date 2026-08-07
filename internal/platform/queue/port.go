package queue

import "context"

type Message struct {
	Topic   string
	Key     string
	Payload []byte
}

type Handler func(ctx context.Context, msg Message) error

type Publisher interface {
	Publish(ctx context.Context, msg Message) error
}

type Subscriber interface {
	Subscribe(ctx context.Context, topic string, h Handler) error
}

// Bus is both publisher and subscriber (convenient for in-process wiring).
type Bus interface {
	Publisher
	Subscriber
	Close() error
}
