package event

import "context"

type Event interface {
	GetEventName() string
}

type EventPublisher interface {
	PublishEvent(ctx context.Context, event Event) error
	PublishRaw(ctx context.Context, topic string, jsonData []byte) error
}
