package customer

import (
	"fmt"

	"customer-service/internal/domain/customer"
)

type EventDispatcher struct {
	handlers map[string]customer.EventHandler
}

func NewEventDispatcher() *EventDispatcher {
	return &EventDispatcher{
		handlers: make(map[string]customer.EventHandler),
	}
}

func (d *EventDispatcher) Register(eventName string, handler customer.EventHandler) {
	d.handlers[eventName] = handler
}

func (d *EventDispatcher) Dispatch(event customer.EventPayload) error {
	handler, ok := d.handlers[event.Name]
	if !ok {
		return fmt.Errorf("no handler registered for event: %s", event.Name)
	}

	return handler.Handle(event)
}
