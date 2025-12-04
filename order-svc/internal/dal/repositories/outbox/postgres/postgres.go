package postgres

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/corray333/backend-labs/order/internal/dal/postgres"
	"github.com/corray333/backend-labs/order/internal/service/models/outbox"
)

// OutboxRepository implements the outbox repository for PostgreSQL.
type OutboxRepository struct {
	client *postgres.Client
}

// NewOutboxRepository creates a new outbox repository.
func NewOutboxRepository(client *postgres.Client) *OutboxRepository {
	return &OutboxRepository{
		client: client,
	}
}

// Insert adds a new message to the outbox.
func (r *OutboxRepository) Insert(ctx context.Context, msg outbox.OutboxMessage) error {
	query, args, err := sq.Insert("outbox").
		Columns(
			"queue_name",
			"exchange_name",
			"routing_key",
			"payload",
			"content_type",
			"retry_count",
			"max_retries",
			"last_error",
			"created_at",
			"updated_at",
			"next_retry_at",
		).
		Values(
			msg.QueueName,
			msg.ExchangeName,
			msg.RoutingKey,
			msg.Payload,
			msg.ContentType,
			msg.RetryCount,
			msg.MaxRetries,
			msg.LastError,
			msg.CreatedAt,
			msg.UpdatedAt,
			msg.NextRetryAt,
		).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert query: %w", err)
	}

	_, err = r.client.DB().ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to insert outbox message: %w", err)
	}

	return nil
}

// GetPendingMessages retrieves messages that are ready for retry.
func (r *OutboxRepository) GetPendingMessages(
	ctx context.Context,
	limit int,
) ([]outbox.OutboxMessage, error) {
	query, args, err := sq.Select(
		"id",
		"queue_name",
		"exchange_name",
		"routing_key",
		"payload",
		"content_type",
		"retry_count",
		"max_retries",
		"last_error",
		"created_at",
		"updated_at",
		"next_retry_at",
	).
		From("outbox").
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
		return nil, fmt.Errorf("failed to query outbox messages: %w", err)
	}
	defer rows.Close()

	var messages []outbox.OutboxMessage
	for rows.Next() {
		var msg outbox.OutboxMessage
		err := rows.Scan(
			&msg.ID,
			&msg.QueueName,
			&msg.ExchangeName,
			&msg.RoutingKey,
			&msg.Payload,
			&msg.ContentType,
			&msg.RetryCount,
			&msg.MaxRetries,
			&msg.LastError,
			&msg.CreatedAt,
			&msg.UpdatedAt,
			&msg.NextRetryAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan outbox message: %w", err)
		}
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating outbox messages: %w", err)
	}

	return messages, nil
}

// Delete removes a message from the outbox after successful delivery.
func (r *OutboxRepository) Delete(ctx context.Context, id int64) error {
	query, args, err := sq.Delete("outbox").
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build delete query: %w", err)
	}

	_, err = r.client.DB().ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete outbox message: %w", err)
	}

	return nil
}

// UpdateRetry updates retry count and error information.
func (r *OutboxRepository) UpdateRetry(
	ctx context.Context,
	id int64,
	retryCount int,
	lastError string,
	nextRetryAt time.Time,
) error {
	query, args, err := sq.Update("outbox").
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
		return fmt.Errorf("failed to update outbox message: %w", err)
	}

	return nil
}
