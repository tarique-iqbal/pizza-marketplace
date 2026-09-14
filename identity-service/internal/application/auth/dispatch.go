package auth

import (
	"context"
	"encoding/json"
	"fmt"

	"identity-service/internal/domain/auth"
	"identity-service/internal/domain/outbox"
	"identity-service/internal/shared/event"
)

func DispatchEventsTx(
	ctx context.Context,
	outboxRepo outbox.OutboxRepository,
	ev *auth.EmailVerification,
) error {
	for _, e := range ev.PullEvents() {
		payload, ok := toEventPayload(e)
		if !ok {
			continue
		}

		body, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal event %s: %w", payload.GetEventName(), err)
		}

		outboxEvent := outbox.NewOutboxEvent(ev.ID, payload.GetEventName(), body)
		if err := outboxRepo.Create(ctx, &outboxEvent); err != nil {
			return fmt.Errorf("failed to create outbox event %s: %w", payload.GetEventName(), err)
		}
	}
	return nil
}

func toEventPayload(e auth.DomainEvent) (event.Event, bool) {
	switch evt := e.(type) {
	case auth.EmailVerificationCreated:
		return newEmailVerificationCreatedPayload(evt), true
	default:
		return nil, false
	}
}
