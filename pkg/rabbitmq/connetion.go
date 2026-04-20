package rabbitmq

import (
	"errors"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
}

type ConnectionManagerImpl struct {
	url        string
	connection *RabbitMQ
	mu         sync.RWMutex
	lastError  error
}

func (c *ConnectionManagerImpl) GetConnection() (*RabbitMQ, error) {
	c.mu.RLock()
	if c.connection != nil && c.IsHealthy() {
		defer c.mu.RUnlock()
		return c.connection, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connection != nil && c.IsHealthy() {
		return c.connection, nil
	}

	rmq, err := NewRabbitMQ(c.url)
	if err != nil {
		c.lastError = err
		return nil, err
	}

	c.connection = rmq
	c.lastError = nil
	return rmq, nil
}

func (c *ConnectionManagerImpl) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connection != nil {
		c.connection.Close()
		c.connection = nil
	}

	return nil
}

func (c *ConnectionManagerImpl) IsHealthy() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.connection == nil || c.connection.Conn == nil {
		return false
	}

	return !c.connection.Conn.IsClosed()
}

func NewConnectionManager(url string) ConnectionManager {
	return &ConnectionManagerImpl{
		url: url,
	}
}

func NewRabbitMQ(url string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, errors.New("Failed to connect to RabbitMQ: " + err.Error())
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, errors.New("Failed to open a channel: " + err.Error())
	}

	return &RabbitMQ{
		Conn:    conn,
		Channel: ch,
	}, nil
}

func (r *RabbitMQ) Close() {
	if r.Channel != nil {
		_ = r.Conn.Close()
	}

	if r.Conn != nil {
		_ = r.Conn.Close()
	}
}
