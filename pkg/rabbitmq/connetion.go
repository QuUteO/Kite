package rabbitmq

import (
	"log/slog"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	Connection *amqp.Connection
	Channel    *amqp.Channel
}

type ConnectionManagerImpl struct {
	url    string
	conn   *RabbitMQ
	mu     sync.Mutex
	logger *slog.Logger
}

func (c *ConnectionManagerImpl) GetConnection() (*RabbitMQ, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil && !c.conn.Connection.IsClosed() {
		return c.conn, nil
	}

	c.logger.Info("establishing new connection to rabbitmq", "url", c.url)

	// ВАЖНО: NewRabbitMQ теперь должна возвращать (*RabbitMQ, error)
	// и не использовать log.Fatalf
	newConn, err := NewRabbitMQ(c.url)
	if err != nil {
		c.logger.Error("failed to connect to rabbitmq", "error", err)
		return nil, err
	}

	c.conn = newConn

	return c.conn, nil
}

func (c *ConnectionManagerImpl) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		if !c.conn.Connection.IsClosed() {
			c.logger.Info("closing connection to rabbitmq", "url", c.url)
			c.conn.Close()
		}
		c.conn = nil
	}

	return nil
}

func (c *ConnectionManagerImpl) IsHealthy() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.conn != nil && !c.conn.Connection.IsClosed()
}

func NewConnectionManager(url string, logger *slog.Logger) ConnectionManager {
	return &ConnectionManagerImpl{
		url:    url,
		logger: logger,
	}
}

func NewRabbitMQ(url string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &RabbitMQ{
		Connection: conn,
		Channel:    ch,
	}, nil
}

func (r *RabbitMQ) Close() {
	if r.Channel != nil {
		r.Channel.Close()
	}

	if r.Connection != nil {
		r.Connection.Close()
	}
}
