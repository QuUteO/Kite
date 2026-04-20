package notification

import (
	"Kite/internal/config"
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

	return &Service{
		cfg:         cfg,
		logger:      logger,
		connManager: connManager,
		consumer:    rabbitmq.NewConsumer(connManager, logger),
	}
}

func (s *Service) Run(ctx context.Context) error {
	s.logger.Info("connecting to rabbitmq")

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
