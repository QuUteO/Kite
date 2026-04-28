package email_sender

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
)

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
	// Настройка TLS
	tlsConfig := &tls.Config{
		ServerName: s.host,
	}

	// Подключаемся с TLS
	conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%s", s.host, s.port), tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	// Создаем SMTP клиент на TLS соединении
	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	// Аутентификация
	auth := smtp.PlainAuth("", s.email, s.password, s.host)
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("failed to auth: %w", err)
	}

	// Отправляем письмо
	msg := []byte("Subject: Notification\r\n\r\n" + body)

	return smtp.SendMail(
		s.host+":"+s.port,
		auth,
		s.email,
		[]string{to},
		msg,
	)
}
