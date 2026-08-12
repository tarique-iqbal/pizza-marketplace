package inbound

import (
	"context"
	"fmt"
	"restaurant-service/internal/domain/restaurant"
)

type EventDispatcher struct {
	handlers map[string]restaurant.EventHandler
}

func NewEventDispatcher() *EventDispatcher {
	return &EventDispatcher{
		handlers: make(map[string]restaurant.EventHandler),
	}
}

func (d *EventDispatcher) Register(eventName string, handler restaurant.EventHandler) {
	d.handlers[eventName] = handler
}

func (d *EventDispatcher) Dispatch(ctx context.Context, event restaurant.EventPayload) error {
	handler, ok := d.handlers[event.Name]
	if !ok {
		return fmt.Errorf("no handler for event: %s", event.Name)
	}
	return handler.Handle(ctx, event)
}
