package rabbitmq

import (
	"fmt"
	"log/slog"

	"github.com/spf13/viper"
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
	host := viper.GetString("rabbitmq.host")
	port := viper.GetInt("rabbitmq.port")
	user := viper.GetString("rabbitmq.user")
	password := viper.GetString("rabbitmq.password")

	if host == "" {
		host = "rabbitmq"
	}
	if port == 0 {
		port = 5672
	}

	connStr := fmt.Sprintf(
		"amqp://%s:%s@%s:%d/",
		user,
		password,
		host,
		port,
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

	slog.Info("RabbitMQ connected", "host", host, "port", port)

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

type ConsumeConfig struct {
	Queue     string
	Consumer  string
	AutoAck   bool
	Exclusive bool
	NoLocal   bool
	NoWait    bool
	Args      amqp.Table
}

// Consume starts consuming messages from the queue.
func (r *Client) Consume(cfg ConsumeConfig) (<-chan amqp.Delivery, error) {
	return r.channel.Consume(
		cfg.Queue,
		cfg.Consumer,
		cfg.AutoAck,
		cfg.Exclusive,
		cfg.NoLocal,
		cfg.NoWait,
		cfg.Args,
	)
}
