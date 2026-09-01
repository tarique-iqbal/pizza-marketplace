package notification

type EventDispatcher interface {
	Register(eventName string, handler EventHandler)
	Dispatch(event EventPayload) error
}
