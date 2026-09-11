package order

import (
	"context"
	"encoding/json"
	"fmt"

	"order-service/internal/domain/order"
	"order-service/internal/domain/outbox"
	"order-service/internal/shared/event"
)

type Enricher func(order.DomainEvent) (event.Event, bool)

func DispatchEventsTx(
	ctx context.Context,
	outboxRepo outbox.OutboxRepository,
	ord *order.Order,
	enrichers ...Enricher,
) error {
	for _, e := range ord.PullEvents() {
		payload, ok := enrich(e, enrichers)
		if !ok {
			continue
		}

		body, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal event %s: %w", payload.GetEventName(), err)
		}

		outboxEvent := outbox.NewOutboxEvent(ord.ID, payload.GetEventName(), body)
		if err := outboxRepo.Create(ctx, &outboxEvent); err != nil {
			return fmt.Errorf("failed to create outbox event %s: %w", payload.GetEventName(), err)
		}
	}

	return nil
}

func enrich(e order.DomainEvent, enrichers []Enricher) (event.Event, bool) {
	for _, fn := range enrichers {
		if payload, ok := fn(e); ok {
			return payload, true
		}
	}

	return nil, false
}
