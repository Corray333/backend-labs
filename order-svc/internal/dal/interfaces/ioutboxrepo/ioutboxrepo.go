package ioutboxrepo

import (
	"context"
	"time"

	"github.com/corray333/backend-labs/order/internal/service/models/outbox"
)

// IOutboxRepository defines the interface for outbox operations.
type IOutboxRepository interface {
	// Insert adds a new message to the outbox
	Insert(ctx context.Context, msg outbox.OutboxMessage) error

	// GetPendingMessages retrieves messages that are ready for retry
	GetPendingMessages(ctx context.Context, limit int) ([]outbox.OutboxMessage, error)

	// Delete removes a message from the outbox after successful delivery
	Delete(ctx context.Context, id int64) error

	// UpdateRetry updates retry count and error information
	UpdateRetry(
		ctx context.Context,
		id int64,
		retryCount int,
		lastError string,
		nextRetryAt time.Time,
	) error
}
