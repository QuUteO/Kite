package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Producer struct {
	connManager ConnectionManager
	logger      *slog.Logger
	mu          sync.Mutex
}

func NewProducer(connManager ConnectionManager, logger *slog.Logger) *Producer {
	return &Producer{
		connManager: connManager,
		logger:      logger,
	}
}

func (p *Producer) PublishToExchange(ctx context.Context, exchange, routingKey string, data interface{}) error {
	rmq, err := p.connManager.GetConnection()
	if err != nil {
		p.logger.Error("failed to connect to exchange")
		return err
	}

	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()
	err = rmq.Channel.PublishWithContext(
		ctx,
		exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish data: %w", err)
	}

	return nil
}
