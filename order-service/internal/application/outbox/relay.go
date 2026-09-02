package outbox

import (
	"context"

	"order-service/internal/domain/outbox"
	"order-service/internal/shared/event"
)

type Relayer interface {
	Process(ctx context.Context, e outbox.OutboxEvent) error
}

type Relay struct {
	publisher event.EventPublisher
}

func NewRelay(publisher event.EventPublisher) *Relay {
	return &Relay{publisher: publisher}
}

func (r *Relay) Process(ctx context.Context, event outbox.OutboxEvent) error {
	routingKey := event.EventName

	return r.publisher.Publish(ctx, routingKey, event.Payload)
}
