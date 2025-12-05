package iinboxrepo

import (
	"context"
	"time"

	"github.com/corray333/backend-labs/consumer/internal/service/models/inbox"
)

// IInboxRepository defines the interface for inbox operations.
type IInboxRepository interface {
	// Insert adds a new message to the inbox
	Insert(ctx context.Context, msg inbox.InboxMessage) error

	// GetPendingMessages retrieves messages that are ready for retry
	GetPendingMessages(ctx context.Context, limit int) ([]inbox.InboxMessage, error)

	// Delete removes a message from the inbox after successful processing
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
