package rabbitmq

import "context"

// ConnectionManager управляет подключением к RabbitMQ
type ConnectionManager interface {
	// GetConnection возвращает активное соединение с каналом
	GetConnection() (*RabbitMQ, error)
	// Close закрывает соединение
	Close() error
	// IsHealthy проверяет состояние подключения
	IsHealthy() bool
}

// Publisher интерфейс для отправки сообщений
type Publisher interface {
	// Publish отправляет сообщение в очередь
	Publish(ctx context.Context, queue string, data interface{}) error
	// PublishToExchange отправляет сообщение в обменник
	PublishToExchange(ctx context.Context, exchange, routingKey string, data interface{}) error
}

// Consumer интерфейс для получения сообщений
type Consumer interface {
	// Subscribe подписывается на очередь и обрабатывает сообщения
	Subscribe(ctx context.Context, queue string, handler MessageHandler) error
	// SubscribeWithExchange подписывается через обменник
	SubscribeWithExchange(ctx context.Context, exchange, queue, routingKey string, handler MessageHandler) error
}

// MessageHandler функция-обработчик сообщений
type MessageHandler func(ctx context.Context, body []byte) error
