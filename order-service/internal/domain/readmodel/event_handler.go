package readmodel

import "context"

type EventHandler interface {
	Handle(ctx context.Context, event EventPayload) error
}

type EventPayload struct {
	Name string
	Data []byte
}
