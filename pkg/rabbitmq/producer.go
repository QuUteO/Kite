package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type ProducerImpl struct {
	connManager ConnectionManager
	queueConfig map[string]QueueConfig
}

type QueueConfig struct {
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	NoWait     bool
}

func NewProducer(connManager ConnectionManager) Publisher {
	return &ProducerImpl{
		connManager: connManager,
		queueConfig: make(map[string]QueueConfig),
	}
}

func (p *ProducerImpl) Publish(ctx context.Context, queue string, data interface{}) error {
	rmq, err := p.connManager.GetConnection()
	if err != nil {
		return errors.New("failed to get connection: " + err.Error())
	}

	body, err := json.Marshal(data)
	if err != nil {
		return errors.New("failed to marshal json: " + err.Error())
	}

	// Получаем или используем дефолтную конфигурацию очереди
	cfg := p.queueConfig[queue]
	if cfg == (QueueConfig{}) {
		cfg = QueueConfig{Durable: true}
	}

	_, err = rmq.Channel.QueueDeclare(
		queue,
		cfg.Durable,
		cfg.AutoDelete,
		cfg.Exclusive,
		cfg.NoWait,
		nil,
	)
	if err != nil {
		return errors.New("failed to declare queue: " + err.Error())
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return rmq.Channel.PublishWithContext(
		ctx,
		"",
		queue,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}

func (p *ProducerImpl) PublishToExchange(ctx context.Context, exchange, routingKey string, data interface{}) error {
	rmq, err := p.connManager.GetConnection()
	if err != nil {
		return errors.New("failed to get connection: " + err.Error())
	}

	body, err := json.Marshal(data)
	if err != nil {
		return errors.New("failed to marshal json: " + err.Error())
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

// SetQueueConfig устанавливает конфигурацию для очереди
func (p *ProducerImpl) SetQueueConfig(queue string, cfg QueueConfig) {
	p.queueConfig[queue] = cfg
}
