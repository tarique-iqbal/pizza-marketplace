package index

import (
	"fmt"

	"search-service/internal/domain/index"
)

type EventDispatcher struct {
	handlers map[string]index.EventHandler
}

func NewEventDispatcher() *EventDispatcher {
	return &EventDispatcher{
		handlers: make(map[string]index.EventHandler),
	}
}

func (d *EventDispatcher) Register(eventName string, handler index.EventHandler) {
	d.handlers[eventName] = handler
}

func (d *EventDispatcher) Dispatch(event index.EventPayload) error {
	handler, ok := d.handlers[event.Name]
	if !ok {
		return fmt.Errorf("no handler registered for event: %s", event.Name)
	}
	return handler.Handle(event)
}
