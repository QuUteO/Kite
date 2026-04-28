package email

import (
	"Kite/pkg/rabbitmq"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	connManager rabbitmq.ConnectionManager
	logger      *slog.Logger
	emailSender Sender
}

func NewConsumer(connManager rabbitmq.ConnectionManager, logger *slog.Logger, emailSender Sender) *Consumer {
	return &Consumer{
		connManager: connManager,
		logger:      logger,
		emailSender: emailSender,
	}
}

func (c *Consumer) SubscribeWithExchange(ctx context.Context, exchange, queue, routingKey string) error {
	go func() {
		for {
			select {
			case <-ctx.Done():
				c.logger.Info("consumer stopped by context")
				return
			default:
				if err := c.consume(ctx, exchange, queue, routingKey); err != nil {
					c.logger.Error("consumer error, retrying in 5s", "error", err)
					time.Sleep(5 * time.Second)
				}
			}
		}
	}()
	return nil
}

func (c *Consumer) consume(ctx context.Context, exchange, queue, routingKey string) error {
	rmq, err := c.connManager.GetConnection()
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	// Setup topology
	if err := c.setupTopology(rmq.Channel, exchange, queue, routingKey); err != nil {
		return fmt.Errorf("setup topology: %w", err)
	}

	// Set QoS
	if err := rmq.Channel.Qos(1, 0, false); err != nil {
		c.logger.Warn("failed to set QoS", "error", err)
	}

	// Start consuming
	msgs, err := rmq.Channel.Consume(
		queue,
		"notification-service-service",
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	c.logger.Info("consumer subscribed", "queue", queue, "exchange", exchange, "routing_key", routingKey)

	// Process messages
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("consumer channel closed")
			}

			if err := c.processMessage(msg); err != nil {
				c.logger.Error("failed to process message", "error", err)
				if err := msg.Nack(false, true); err != nil {
					c.logger.Error("failed to nack message", "error", err)
				}
			} else {
				if err := msg.Ack(false); err != nil {
					c.logger.Error("failed to ack message", "error", err)
				}
			}
		}
	}
}

func (c *Consumer) setupTopology(ch *amqp.Channel, exchange, queue, routingKey string) error {
	// Declare exchange
	if err := ch.ExchangeDeclare(
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

	// Declare queue
	q, err := ch.QueueDeclare(
		queue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}

	c.logger.Info("queue declared", "name", q.Name, "messages", q.Messages)

	// Bind queue
	if err := ch.QueueBind(q.Name, routingKey, exchange, false, nil); err != nil {
		return fmt.Errorf("bind queue: %w", err)
	}

	return nil
}

func (c *Consumer) processMessage(msg amqp.Delivery) error {
	var emailMsg struct {
		Email string `json:"email"`
		Value int    `json:"value"`
	}

	if err := json.Unmarshal(msg.Body, &emailMsg); err != nil {
		return fmt.Errorf("unmarshal message: %w", err)
	}

	if emailMsg.Email == "" {
		return fmt.Errorf("email address is empty")
	}

	body := fmt.Sprintf("Your verification code is: %d", emailMsg.Value)

	c.logger.Info("sending email", "to", emailMsg.Email, "code", emailMsg.Value)

	if err := c.emailSender.Send(emailMsg.Email, body); err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	c.logger.Info("email sent successfully", "to", emailMsg.Email)
	return nil
}
