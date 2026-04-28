package email

import (
	"fmt"
	"net/smtp"
)

type Sender interface {
	Send(to string, body string) error
}

type SMTPSender struct {
	host     string
	port     string
	email    string
	password string
}

func NewSMTPSender(host, port, email, password string) *SMTPSender {
	return &SMTPSender{
		host:     host,
		port:     port,
		email:    email,
		password: password,
	}
}

func (s *SMTPSender) Send(to string, body string) error {
	auth := smtp.PlainAuth("", s.email, s.password, s.host)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: Notification\r\n\r\n%s",
		s.email, to, body)

	addr := fmt.Sprintf("%s:%s", s.host, s.port)

	if err := smtp.SendMail(addr, auth, s.email, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("send mail: %w", err)
	}

	return nil
}
