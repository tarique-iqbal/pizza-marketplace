package notification

type Message struct {
	To      string
	Subject string // channel-specific: used by email, ignored by channels that don't have one
	Body    string
}

type Sender interface {
	Send(msg Message) error
}
