package notification

import (
	"Kite/internal/config"
	"Kite/internal/notification"
	"Kite/pkg/rabbitmq"
	"context"
	"fmt"
	"log/slog"
)

type Service struct {
	cfg         *config.Config
	logger      *slog.Logger
	connManager rabbitmq.ConnectionManager
	consumer    rabbitmq.Consumer
}

func NewNotification(cfg *config.Config, logger *slog.Logger) *Service {
	connManager := rabbitmq.NewConnectionManager(cfg.RabbitMQ.URL, logger)

	emailSender := email_sender.NewSMTPSender(
		cfg.SMTP.Host,
		cfg.SMTP.Port,
		cfg.SMTP.From,
		cfg.SMTP.Password,
	)

	return &Service{
		cfg:         cfg,
		logger:      logger,
		connManager: connManager,
		consumer:    email_sender.NewConsumer(connManager, logger, emailSender),
	}
}

func (s *Service) Run(ctx context.Context) error {
	_, err := s.connManager.GetConnection()
	if err != nil {
		return fmt.Errorf("rabbitmq connection failed: %w", err)
	}

	s.logger.Info("connected to rabbitmq")

	err = s.consumer.SubscribeWithExchange(
		ctx,
		s.cfg.RabbitMQ.Exchange,
		s.cfg.RabbitMQ.EmailRegisterQueue,
		s.cfg.RabbitMQ.EmailLoginRoutingKey,
	)
	if err != nil {
		return err
	}

	<-ctx.Done()

	s.logger.Info("shutting down notification service")

	return s.connManager.Close()
}
