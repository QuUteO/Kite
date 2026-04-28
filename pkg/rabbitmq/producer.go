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
		return err
	}

	err = rmq.Channel.ExchangeDeclare(
		exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("exchange declare failed: %w", err)
	}

	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return rmq.Channel.PublishWithContext(
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
}
