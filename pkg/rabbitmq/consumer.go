package rabbitmq

import (
	"context"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitConsumer struct {
	connManager ConnectionManager
	logger      *slog.Logger
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

				// 3. Подписываемся
				msgs, err := rmq.Channel.Consume(
					queue,
					"",    // consumer tag
					false, // auto-ack (лучше false для надежности)
					false, // exclusive
					false, // no-local
					false, // no-wait
					nil,   // args
				)
				if err != nil {
					r.logger.Error("failed to consume", "error", err)
					time.Sleep(5 * time.Second)
					continue
				}

				r.logger.Info("consumer subscribed", "queue", queue, "exchange", exchange)

				// 4. Читаем сообщения
				for d := range msgs {
					r.logger.Info("message received", "body", string(d.Body))

					if err := d.Ack(false); err != nil {
						r.logger.Error("failed to ack message", "error", err)
					}
				}

				r.logger.Warn("channel closed, reconnecting...")
			}
		}
	}()

	return nil
}

func (r *RabbitConsumer) setupTopology(ch *amqp.Channel, exchange, queue, routingKey string) error {
	// Объявляем обменник
	err := ch.ExchangeDeclare(
		exchange,
		"topic", // или "direct" в зависимости от задач
		true,    // durable
		false,   // auto-deleted
		false,   // internal
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		return err
	}

	// Объявляем очередь
	_, err = ch.QueueDeclare(
		queue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return err
	}

	// Связываем их
	return ch.QueueBind(queue, routingKey, exchange, false, nil)
}

func NewConsumer(connManager ConnectionManager, logger *slog.Logger) Consumer {
	return &RabbitConsumer{
		connManager: connManager,
		logger:      logger,
	}
}
