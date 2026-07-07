package rabbitmq

import (
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type Connection struct {
	url            string
	logger         *zap.Logger
	conn           *amqp.Connection
	mu             sync.RWMutex
	isClosed       bool
	reconnectDelay time.Duration
	pool           chan *amqp.Channel
}

func NewConnection(url string, logger *zap.Logger) (*Connection, error) {
	c := &Connection{
		url:            url,
		logger:         logger,
		reconnectDelay: 2 * time.Second,
		pool:           make(chan *amqp.Channel, 100),
	}

	if err := c.connect(); err != nil {
		return nil, err
	}

	go c.reconnectLoop()

	return c, nil
}

func (c *Connection) connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, err := amqp.Dial(c.url)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	c.conn = conn

	// Clear old pool channels if any
	for {
		select {
		case ch := <-c.pool:
			if ch != nil {
				ch.Close()
			}
		default:
			return nil
		}
	}
}

func (c *Connection) reconnectLoop() {
	for {
		c.mu.RLock()
		isClosed := c.isClosed
		c.mu.RUnlock()

		if isClosed {
			return
		}

		c.mu.RLock()
		connClosed := make(chan *amqp.Error, 1)
		if c.conn != nil {
			c.conn.NotifyClose(connClosed)
		}
		c.mu.RUnlock()

		err := <-connClosed
		if err != nil {
			c.logger.Error("RabbitMQ connection closed with error", zap.Error(err))
		} else {
			c.logger.Warn("RabbitMQ connection notify channel closed")
		}

		c.mu.RLock()
		isClosed = c.isClosed
		c.mu.RUnlock()

		if isClosed {
			return // Intended shutdown
		}

		c.logger.Info("Attempting to reconnect to RabbitMQ...")
		for {
			c.mu.RLock()
			isClosed := c.isClosed
			c.mu.RUnlock()

			if isClosed {
				return
			}

			if err := c.connect(); err != nil {
				c.logger.Warn("Failed to reconnect, retrying...", zap.Error(err), zap.Duration("delay", c.reconnectDelay))
				time.Sleep(c.reconnectDelay)
				continue
			}

			c.logger.Info("Successfully reconnected to RabbitMQ")
			break
		}
	}
}

func (c *Connection) AcquireChannel() (*amqp.Channel, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conn == nil || c.conn.IsClosed() {
		return nil, fmt.Errorf("connection is not open")
	}

	select {
	case ch := <-c.pool:
		if ch.IsClosed() {
			return c.createNewChannel()
		}
		return ch, nil
	default:
		return c.createNewChannel()
	}
}

func (c *Connection) createNewChannel() (*amqp.Channel, error) {
	ch, err := c.conn.Channel()
	if err != nil {
		return nil, err
	}
	if err := ch.Confirm(false); err != nil {
		ch.Close()
		return nil, err
	}
	return ch, nil
}

func (c *Connection) ReleaseChannel(ch *amqp.Channel) {
	if ch == nil || ch.IsClosed() {
		return
	}
	select {
	case c.pool <- ch:
	default:
		ch.Close() // Pool is full
	}
}

func (c *Connection) SetupExchange(exchangeName string) error {
	ch, err := c.AcquireChannel()
	if err != nil {
		return err
	}

	err = ch.ExchangeDeclare(
		exchangeName, // name
		"topic",      // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		ch.Close()
		return fmt.Errorf("failed to declare exchange: %w", err)
	}
	c.ReleaseChannel(ch)

	return nil
}

func (c *Connection) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.isClosed = true

	for {
		select {
		case ch := <-c.pool:
			if ch != nil {
				ch.Close()
			}
		default:
			goto PoolEmpty
		}
	}
PoolEmpty:
	if c.conn != nil {
		c.conn.Close()
	}
}
