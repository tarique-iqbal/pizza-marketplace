package notification

import "context"

type EventDispatcher interface {
	Register(eventName string, handler EventHandler)
	Dispatch(ctx context.Context, event EventPayload) error
}
