package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Producer struct {
	connManager ConnectionManager
	logger      *slog.Logger
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
		return fmt.Errorf("get connection: %w", err)
	}

	// Declare exchange
	if err := rmq.Channel.ExchangeDeclare(
		exchange,
		"topic",
		true,  // durable
		false, // auto-delete
		false, // internal
		false, // no-wait
		nil,   // args
	); err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}

	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := rmq.Channel.PublishWithContext(
		ctx,
		exchange,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		},
	); err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	p.logger.Debug("message published",
		"exchange", exchange,
		"routing_key", routingKey)
	return nil
}
