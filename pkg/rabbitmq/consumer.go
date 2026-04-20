package rabbitmq

import (
	"context"
	"errors"
)

type ConsumerImpl struct {
	connManager ConnectionManager
}

func NewConsumer(connManager ConnectionManager) Consumer {
	return &ConsumerImpl{
		connManager: connManager,
	}
}

func (c *ConsumerImpl) Subscribe(ctx context.Context, queue string, handler MessageHandler) error {
	rmq, err := c.connManager.GetConnection()
	if err != nil {
		return errors.New("failed to get connection: " + err.Error())
	}

	_, err = rmq.Channel.QueueDeclare(queue, true, false, false, false, nil)
	if err != nil {
		return errors.New("failed to declare a queue: " + err.Error())
	}

	msgs, err := rmq.Channel.Consume(queue, "", false, false, false, false, nil)
	if err != nil {
		return errors.New("failed to consume a queue: " + err.Error())
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					return
				}

				if err := handler(ctx, msg.Body); err == nil {
					msg.Ack(false)
				} else {
					msg.Nack(false, true)
				}
			}
		}
	}()

	return nil
}

func (c *ConsumerImpl) SubscribeWithExchange(ctx context.Context, exchange, queue, routingKey string, handler MessageHandler) error {
	rmq, err := c.connManager.GetConnection()
	if err != nil {
		return errors.New("failed to get connection: " + err.Error())
	}

	// Объявляем обменник
	err = rmq.Channel.ExchangeDeclare(exchange, "topic", true, false, false, false, nil)
	if err != nil {
		return errors.New("failed to declare exchange: " + err.Error())
	}

	// Объявляем очередь
	_, err = rmq.Channel.QueueDeclare(queue, true, false, false, false, nil)
	if err != nil {
		return errors.New("failed to declare queue: " + err.Error())
	}

	// Привязываем очередь к обменнику
	err = rmq.Channel.QueueBind(queue, routingKey, exchange, false, nil)
	if err != nil {
		return errors.New("failed to bind queue: " + err.Error())
	}

	// Начинаем потребление
	msgs, err := rmq.Channel.Consume(queue, "", false, false, false, false, nil)
	if err != nil {
		return errors.New("failed to consume: " + err.Error())
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					return
				}

				if err := handler(ctx, msg.Body); err == nil {
					msg.Ack(false)
				} else {
					msg.Nack(false, true)
				}
			}
		}
	}()

	return nil
}
