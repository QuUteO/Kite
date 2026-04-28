package notification

import (
	"Kite/internal/config"
	"Kite/internal/notification/email"
	"Kite/pkg/rabbitmq"
	"context"
	"fmt"
	"log/slog"
)

type Service struct {
	cfg         *config.Config
	logger      *slog.Logger
	connManager rabbitmq.ConnectionManager
	consumer    *email.Consumer
}

func NewService(cfg *config.Config, logger *slog.Logger) *Service {
	connManager := rabbitmq.NewConnectionManager(cfg.RabbitMQ.URL, logger)
	emailSender := email.NewSMTPSender(
		cfg.SMTP.Host,
		cfg.SMTP.Port,
		cfg.SMTP.From,
		cfg.SMTP.Password,
	)

	return &Service{
		cfg:         cfg,
		logger:      logger,
		connManager: connManager,
		consumer:    email.NewConsumer(connManager, logger, emailSender),
	}
}

func (s *Service) Run(ctx context.Context) error {
	// Test connection
	if _, err := s.connManager.GetConnection(); err != nil {
		return fmt.Errorf("rabbitmq connection failed: %w", err)
	}

	s.logger.Info("connected to rabbitmq",
		"url", s.cfg.RabbitMQ.URL,
		"exchange", s.cfg.RabbitMQ.Exchange,
		"queue", s.cfg.RabbitMQ.RegisterQueue,
		"routing_key", s.cfg.RabbitMQ.RegisterRoutingKey)

	// Start consuming messages
	if err := s.consumer.SubscribeWithExchange(
		ctx,
		s.cfg.RabbitMQ.Exchange,
		s.cfg.RabbitMQ.RegisterQueue,
		s.cfg.RabbitMQ.RegisterRoutingKey,
	); err != nil {
		return fmt.Errorf("subscribe failed: %w", err)
	}

	s.logger.Info("notification-service service started, waiting for messages")

	<-ctx.Done()

	s.logger.Info("shutting down notification-service service")
	return s.connManager.Close()
}
