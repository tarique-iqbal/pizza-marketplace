package restaurant

import (
	"context"

	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/shared/event"
)

func DispatchEvents(ctx context.Context, publisher event.EventPublisher, res *restaurant.Restaurant) {
	for _, e := range res.PullEvents() {
		payload, ok := toEventPayload(e)
		if !ok {
			continue
		}

		_ = publisher.PublishEvent(ctx, payload)
	}
}

func toEventPayload(e restaurant.DomainEvent) (event.Event, bool) {
	switch evt := e.(type) {
	case restaurant.RestaurantReadyForReview:
		payload := RestaurantReadyForReviewPayload{
			RestaurantID:   evt.RestaurantID,
			RestaurantName: evt.RestaurantName,
			ReadyAt:        evt.ReadyAt,
		}
		payload.EventName = payload.GetEventName()

		return payload, true
	default:
		return nil, false
	}
}
