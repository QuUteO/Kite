package rabbitmq

import "context"

// ConnectionManager управляет подключением к RabbitMQ
type ConnectionManager interface {
	GetConnection() (*RabbitMQ, error)
	Close() error
	IsHealthy() bool
}

// Publisher интерфейс для отправки сообщений
type Publisher interface {
	PublishToExchange(ctx context.Context, exchange, routingKey string, data interface{}) error
}

// Consumer интерфейс для получения сообщений
type Consumer interface {
	SubscribeWithExchange(ctx context.Context, exchange, queue, routingKey string) error
}
