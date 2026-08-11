package testutil

import (
	"context"

	"restaurant-service/internal/shared/event"
)

type NoopPublisher struct{}

func (NoopPublisher) PublishEvent(ctx context.Context, e event.Event) error {
	return nil
}
