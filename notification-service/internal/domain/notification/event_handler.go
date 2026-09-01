package notification

type EventHandler interface {
	Handle(event EventPayload) error
}

type EventPayload struct {
	Name string
	Data []byte
}
