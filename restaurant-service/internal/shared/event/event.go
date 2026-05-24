package event

import "context"

type Event interface {
	GetEventName() string
}

type EventPublisher interface {
	PublishEvent(ctx context.Context, event Event) error
}
