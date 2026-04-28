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

func (r *RabbitMQ) Close() error {
	var err error
	if r.Channel != nil {
		if cerr := r.Channel.Close(); cerr != nil {
			err = cerr
		}
	}
	if r.Connection != nil {
		if cerr := r.Connection.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}
	return err
}

type ConnectionManagerImpl struct {
	url    string
	conn   *RabbitMQ
	mu     sync.RWMutex
	logger *slog.Logger
}

func NewConnectionManager(url string, logger *slog.Logger) ConnectionManager {
	return &ConnectionManagerImpl{
		url:    url,
		logger: logger,
	}
}

func (m *ConnectionManagerImpl) GetConnection() (*RabbitMQ, error) {
	m.mu.RLock()
	if m.conn != nil && m.conn.Connection != nil && !m.conn.Connection.IsClosed() {
		defer m.mu.RUnlock()
		return m.conn, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double check after acquiring write lock
	if m.conn != nil && m.conn.Connection != nil && !m.conn.Connection.IsClosed() {
		return m.conn, nil
	}

	m.logger.Info("establishing new connection to rabbitmq", "url", m.url)

	conn, err := amqp.Dial(m.url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	m.conn = &RabbitMQ{
		Connection: conn,
		Channel:    ch,
	}

	return m.conn, nil
}

func (m *ConnectionManagerImpl) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.conn != nil {
		m.logger.Info("closing rabbitmq connection")
		if err := m.conn.Close(); err != nil {
			m.logger.Error("error closing rabbitmq connection", "error", err)
			return err
		}
		m.conn = nil
	}
	return nil
}

func (m *ConnectionManagerImpl) IsHealthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.conn != nil && m.conn.Connection != nil && !m.conn.Connection.IsClosed()
}
