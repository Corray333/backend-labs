package outbox

import (
	"time"
)

// OutboxMessage represents a message that failed to be published to RabbitMQ.
type OutboxMessage struct {
	ID           int64
	QueueName    string
	ExchangeName string
	RoutingKey   string
	Payload      []byte
	ContentType  string
	RetryCount   int
	MaxRetries   int
	LastError    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	NextRetryAt  time.Time
}
