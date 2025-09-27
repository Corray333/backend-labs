package rabbitmq

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/streadway/amqp"
)

// Client represents a RabbitMQ client.
type Client struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// Channel returns the underlying AMQP channel.
func (r *Client) Channel() *amqp.Channel {
	return r.channel
}

// Connection returns the underlying AMQP connection.
func (r *Client) Connection() *amqp.Connection {
	return r.conn
}

// Close closes the channel and connection for graceful shutdown.
func (r *Client) Close() error {
	if r.channel != nil {
		if err := r.channel.Close(); err != nil {
			return err
		}
	}
	if r.conn != nil {
		return r.conn.Close()
	}

	return nil
}

// MustNewClient creates a new RabbitMQ client.
func MustNewClient() *Client {
	connStr := fmt.Sprintf(
		"amqp://%s:%s@%s:5672/",
		os.Getenv("RABBITMQ_DEFAULT_USER"),
		os.Getenv("RABBITMQ_DEFAULT_PASS"),
		"rabbitmq",
	)

	conn, err := amqp.Dial(connStr)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to RabbitMQ: %v", err))
	}

	channel, err := conn.Channel()
	if err != nil {
		err := conn.Close()
		if err != nil {
			panic(fmt.Sprintf("Failed to close a connection: %v", err))
		}
		panic(fmt.Sprintf("Failed to open a channel: %v", err))
	}

	slog.Info("RabbitMQ connected")

	return &Client{
		conn:    conn,
		channel: channel,
	}
}

type DeclareQueueConfig struct {
	Name       string
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	NoWait     bool
	Args       amqp.Table
}

// DeclareQueue declares a queue with the given configuration.
func (r *Client) DeclareQueue(cfg DeclareQueueConfig) (amqp.Queue, error) {
	return r.channel.QueueDeclare(
		cfg.Name,
		cfg.Durable,
		cfg.AutoDelete,
		cfg.Exclusive,
		cfg.NoWait,
		cfg.Args,
	)
}
