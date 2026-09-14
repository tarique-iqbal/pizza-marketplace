package user

import (
	"context"
	"encoding/json"
	"fmt"

	"identity-service/internal/domain/outbox"
	"identity-service/internal/domain/user"
	"identity-service/internal/shared/event"

	"github.com/google/uuid"
)

func DispatchEventsTx(ctx context.Context, outboxRepo outbox.OutboxRepository, u *user.User) error {
	for _, e := range u.PullEvents() {
		payload, aggregateID, ok := toEventPayload(e)
		if !ok {
			continue
		}

		body, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal event %s: %w", payload.GetEventName(), err)
		}

		outboxEvent := outbox.NewOutboxEvent(aggregateID, payload.GetEventName(), body)
		if err := outboxRepo.Create(ctx, &outboxEvent); err != nil {
			return fmt.Errorf("failed to create outbox event %s: %w", payload.GetEventName(), err)
		}
	}
	return nil
}

func toEventPayload(e user.DomainEvent) (event.Event, uuid.UUID, bool) {
	switch evt := e.(type) {
	case user.UserRegistered:
		return newUserRegisteredPayload(evt), evt.UserID, true
	case user.RestaurantInitiated:
		return newRestaurantInitiatedPayload(evt), evt.RestaurantID, true
	default:
		return nil, uuid.Nil, false
	}
}
