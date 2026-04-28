package email_sender

type EmailSender interface {
	Send(to string, body string) error
}
