package email

import (
	"fmt"
	"net/smtp"
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

func (s *SMTPSender) SendEmail(to, subject, body string) error {
	auth := smtp.PlainAuth("", s.username, s.password, s.host)

	msg := []byte(
		"From: " + s.from + "\r\n" +
			"To: " + to + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
			"Content-Transfer-Encoding: 8bit\r\n" +
			"\r\n" + body + "\r\n",
	)

	addr := fmt.Sprintf("%s:%s", s.host, s.port)

	err := smtp.SendMail(addr, auth, s.from, []string{to}, msg)
	if err != nil {
		return err
	}

	return nil
}
