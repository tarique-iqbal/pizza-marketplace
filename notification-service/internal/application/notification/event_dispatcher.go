package notification

import (
	"fmt"
	"notification-service/internal/domain/notification"
)

type EventDispatcher struct {
	handlers map[string]notification.EventHandler
}

func NewEventDispatcher() *EventDispatcher {
	return &EventDispatcher{
		handlers: make(map[string]notification.EventHandler),
	}
}

func (d *EventDispatcher) Register(eventName string, handler notification.EventHandler) {
	d.handlers[eventName] = handler
}

func (d *EventDispatcher) Dispatch(event notification.EventPayload) error {
	handler, ok := d.handlers[event.Name]
	if !ok {
		return fmt.Errorf("no handler registered for event: %s", event.Name)
	}
	return handler.Handle(event)
}
