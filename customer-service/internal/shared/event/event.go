package event

import "context"

type Event interface {
	GetEventName() string
}

type EventPublisher interface {
	Publish(ctx context.Context, routingKey string, payload []byte) error
}
