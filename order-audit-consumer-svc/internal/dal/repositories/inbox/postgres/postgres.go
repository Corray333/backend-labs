package postgres

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/corray333/backend-labs/consumer/internal/dal/postgres"
	"github.com/corray333/backend-labs/consumer/internal/service/models/inbox"
)

// InboxRepository implements the inbox repository for PostgreSQL.
type InboxRepository struct {
	client *postgres.Client
}

// NewInboxRepository creates a new inbox repository.
func NewInboxRepository(client *postgres.Client) *InboxRepository {
	return &InboxRepository{
		client: client,
	}
}

// Insert adds a new message to the inbox.
func (r *InboxRepository) Insert(ctx context.Context, msg inbox.InboxMessage) error {
	query, args, err := sq.Insert("inbox").
		Columns(
			"message_id",
			"queue_name",
			"routing_key",
			"payload",
			"content_type",
			"retry_count",
			"max_retries",
			"last_error",
			"created_at",
			"updated_at",
			"next_retry_at",
			"delivery_tag",
		).
		Values(
			msg.MessageID,
			msg.QueueName,
			msg.RoutingKey,
			msg.Payload,
			msg.ContentType,
			msg.RetryCount,
			msg.MaxRetries,
			msg.LastError,
			msg.CreatedAt,
			msg.UpdatedAt,
			msg.NextRetryAt,
			msg.DeliveryTag,
		).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert query: %w", err)
	}

	_, err = r.client.DB().ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to insert inbox message: %w", err)
	}

	return nil
}

// GetPendingMessages retrieves messages that are ready for retry.
func (r *InboxRepository) GetPendingMessages(
	ctx context.Context,
	limit int,
) ([]inbox.InboxMessage, error) {
	query, args, err := sq.Select(
		"id",
		"message_id",
		"queue_name",
		"routing_key",
		"payload",
		"content_type",
		"retry_count",
		"max_retries",
		"last_error",
		"created_at",
		"updated_at",
		"next_retry_at",
		"delivery_tag",
	).
		From("inbox").
		Where(sq.LtOrEq{"next_retry_at": time.Now()}).
		Where(sq.Expr("retry_count < max_retries")).
		OrderBy("next_retry_at ASC").
		Limit(uint64(limit)).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	rows, err := r.client.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query inbox messages: %w", err)
	}
	defer rows.Close()

	var messages []inbox.InboxMessage
	for rows.Next() {
		var msg inbox.InboxMessage
		err := rows.Scan(
			&msg.ID,
			&msg.MessageID,
			&msg.QueueName,
			&msg.RoutingKey,
			&msg.Payload,
			&msg.ContentType,
			&msg.RetryCount,
			&msg.MaxRetries,
			&msg.LastError,
			&msg.CreatedAt,
			&msg.UpdatedAt,
			&msg.NextRetryAt,
			&msg.DeliveryTag,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inbox message: %w", err)
		}
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating inbox messages: %w", err)
	}

	return messages, nil
}

// Delete removes a message from the inbox after successful processing.
func (r *InboxRepository) Delete(ctx context.Context, id int64) error {
	query, args, err := sq.Delete("inbox").
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build delete query: %w", err)
	}

	_, err = r.client.DB().ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete inbox message: %w", err)
	}

	return nil
}

// UpdateRetry updates retry count and error information.
func (r *InboxRepository) UpdateRetry(
	ctx context.Context,
	id int64,
	retryCount int,
	lastError string,
	nextRetryAt time.Time,
) error {
	query, args, err := sq.Update("inbox").
		Set("retry_count", retryCount).
		Set("last_error", lastError).
		Set("next_retry_at", nextRetryAt).
		Set("updated_at", time.Now()).
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	_, err = r.client.DB().ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update inbox message: %w", err)
	}

	return nil
}
