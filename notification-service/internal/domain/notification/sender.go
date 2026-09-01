package notification

type Sender interface {
	SendEmail(to, subject, body string) error
}
