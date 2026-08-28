package restaurant

import (
	"context"
	"encoding/json"
	"fmt"

	"restaurant-service/internal/domain/outbox"
	"restaurant-service/internal/domain/restaurant"
	logobs "restaurant-service/internal/infrastructure/observability/logger"
	"restaurant-service/internal/shared/event"
)

var outboxRoutingKeys = map[string]bool{
	"restaurant.launched":               true,
	"restaurant.updated":                true,
	"restaurant.pizza_updated":          true,
	"restaurant.topping_prices_updated": true,
}

type Enricher func(restaurant.DomainEvent) (event.Event, bool)

func DispatchEvents(
	ctx context.Context,
	publisher event.EventPublisher,
	res *restaurant.Restaurant,
	enrichers ...Enricher,
) {
	for _, e := range res.PullEvents() {
		payload, ok := enrich(e, enrichers)
		if !ok {
			payload, ok = toEventPayload(e)
		}
		if !ok {
			continue
		}

		if err := publisher.PublishEvent(ctx, payload); err != nil {
			logobs.FromContext(ctx).Warn(
				"failed to publish event", "event", payload.GetEventName(), "error", err,
			)
		}
	}
}

func DispatchEventsTx(
	ctx context.Context,
	outboxRepo outbox.OutboxRepository,
	res *restaurant.Restaurant,
	enrichers ...Enricher,
) ([]event.Event, error) {
	var bestEffort []event.Event

	for _, e := range res.PullEvents() {
		payload, ok := enrich(e, enrichers)
		if !ok {
			payload, ok = toEventPayload(e)
		}
		if !ok {
			continue
		}

		if !outboxRoutingKeys[payload.GetEventName()] {
			bestEffort = append(bestEffort, payload)
			continue
		}

		body, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal event %s: %w", payload.GetEventName(), err)
		}

		outboxEvent := outbox.NewOutboxEvent(res.ID, payload.GetEventName(), body)
		if err := outboxRepo.Create(ctx, &outboxEvent); err != nil {
			return nil, fmt.Errorf("failed to create outbox event %s: %w", payload.GetEventName(), err)
		}
	}

	return bestEffort, nil
}

func PublishBestEffort(ctx context.Context, publisher event.EventPublisher, events []event.Event) {
	for _, payload := range events {
		if err := publisher.PublishEvent(ctx, payload); err != nil {
			logobs.FromContext(ctx).Warn(
				"failed to publish event", "event", payload.GetEventName(), "error", err,
			)
		}
	}
}

func enrich(e restaurant.DomainEvent, enrichers []Enricher) (event.Event, bool) {
	for _, fn := range enrichers {
		if payload, ok := fn(e); ok {
			return payload, true
		}
	}

	return nil, false
}

func toEventPayload(e restaurant.DomainEvent) (event.Event, bool) {
	switch evt := e.(type) {
	case restaurant.RestaurantReadyForReview:
		return newRestaurantReadyForReviewPayload(evt), true
	case restaurant.RestaurantApproved:
		return newRestaurantApprovedPayload(evt), true
	default:
		return nil, false
	}
}
