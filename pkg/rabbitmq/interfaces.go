package rabbitmq

import "context"

type ConnectionManager interface {
	GetConnection() (*RabbitMQ, error)
	Close() error
	IsHealthy() bool
}

type Publisher interface {
	PublishToExchange(ctx context.Context, exchange, routingKey string, data interface{}) error
}

type Consumer interface {
	SubscribeWithExchange(ctx context.Context, exchange, queue, routingKey string) error
}
