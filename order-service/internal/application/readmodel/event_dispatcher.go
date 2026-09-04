package readmodel

import (
	"fmt"

	"order-service/internal/domain/readmodel"
)

type EventDispatcher struct {
	handlers map[string]readmodel.EventHandler
}

func NewEventDispatcher() *EventDispatcher {
	return &EventDispatcher{
		handlers: make(map[string]readmodel.EventHandler),
	}
}

func (d *EventDispatcher) Register(eventName string, handler readmodel.EventHandler) {
	d.handlers[eventName] = handler
}

func (d *EventDispatcher) Dispatch(event readmodel.EventPayload) error {
	handler, ok := d.handlers[event.Name]
	if !ok {
		return fmt.Errorf("no handler registered for event: %s", event.Name)
	}

	return handler.Handle(event)
}
