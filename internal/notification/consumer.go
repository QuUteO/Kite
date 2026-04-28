package email_sender

import (
	"Kite/pkg/rabbitmq"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitConsumer struct {
	connManager rabbitmq.ConnectionManager
	logger      *slog.Logger
	emailSender EmailSender
}

func (r *RabbitConsumer) SubscribeWithExchange(ctx context.Context, exchange, queue, routingKey string) error {
	go func() {
		for {
			select {
			case <-ctx.Done():
				r.logger.Info("consumer stopped by context")
				return
			default:
				// 1. Получаем соединение
				rmq, err := r.connManager.GetConnection()
				if err != nil {
					r.logger.Error("failed to get connection", "error", err)
					time.Sleep(5 * time.Second)
					continue
				}

				err = r.setupTopology(rmq.Channel, exchange, queue, routingKey)
				if err != nil {
					r.logger.Error("failed to setup topology", "error", err)
					time.Sleep(5 * time.Second)
					continue
				}

				msgs, err := rmq.Channel.Consume(
					queue,
					"",
					false,
					false,
					false,
					false,
					nil,
				)
				if err != nil {
					r.logger.Error("failed to consume", "error", err)
					time.Sleep(5 * time.Second)
					continue
				}

				r.logger.Info("consumer subscribed", "queue", queue, "exchange", exchange)

				// 4. Читаем сообщения
				for d := range msgs {
					var msg struct {
						Email string `json:"email"`
						Value int    `json:"value"`
					}

					if err := json.Unmarshal(d.Body, &msg); err != nil {
						r.logger.Error("invalid message", "error", err)
						d.Nack(false, false)
						continue
					}

					body := fmt.Sprintf("Your random number is: %d", msg.Value)

					if err := r.emailSender.Send(msg.Email, body); err != nil {
						r.logger.Error("failed to send email", "error", err)
						d.Nack(false, true)
						continue
					}

					r.logger.Info("email sent", "to", msg.Email)

					d.Ack(false)
				}
			}
		}
	}()

	return nil
}

func (r *RabbitConsumer) setupTopology(ch *amqp.Channel, exchange, queue, routingKey string) error {
	err := ch.ExchangeDeclare(
		exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	_, err = ch.QueueDeclare(
		queue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	return ch.QueueBind(queue, routingKey, exchange, false, nil)
}

func NewConsumer(connManager rabbitmq.ConnectionManager, logger *slog.Logger, sender EmailSender) rabbitmq.Consumer {
	return &RabbitConsumer{
		connManager: connManager,
		logger:      logger,
		emailSender: sender,
	}
}
