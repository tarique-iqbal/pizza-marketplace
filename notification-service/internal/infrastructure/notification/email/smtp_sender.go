package email

import (
	"fmt"
	"net/smtp"

	"notification-service/internal/domain/notification"
)

type SMTPSender struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func NewSMTPSender(host, port, username, password, from string) *SMTPSender {
	return &SMTPSender{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

func (s *SMTPSender) Send(msg notification.Message) error {
	auth := smtp.PlainAuth("", s.username, s.password, s.host)

	raw := []byte(
		"From: " + s.from + "\r\n" +
			"To: " + msg.To + "\r\n" +
			"Subject: " + msg.Subject + "\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
			"Content-Transfer-Encoding: 8bit\r\n" +
			"\r\n" + msg.Body + "\r\n",
	)

	addr := fmt.Sprintf("%s:%s", s.host, s.port)

	err := smtp.SendMail(addr, auth, s.from, []string{msg.To}, raw)
	if err != nil {
		return err
	}

	return nil
}
