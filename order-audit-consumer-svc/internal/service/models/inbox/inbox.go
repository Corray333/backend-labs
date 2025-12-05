package inbox

import (
	"time"
)

// InboxMessage represents a message that failed to be processed.
type InboxMessage struct {
	ID          int64
	MessageID   string
	QueueName   string
	RoutingKey  string
	Payload     []byte
	ContentType string
	RetryCount  int
	MaxRetries  int
	LastError   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	NextRetryAt time.Time
	DeliveryTag uint64
}
