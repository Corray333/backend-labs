package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/corray333/backend-labs/order/internal/dal/interfaces/ioutboxrepo"
	"github.com/corray333/backend-labs/order/internal/dal/postgres"
	"github.com/corray333/backend-labs/order/internal/dal/rabbitmq"
	"github.com/corray333/backend-labs/order/internal/service/models/auditlog"
	"github.com/corray333/backend-labs/order/internal/service/models/order"
	"github.com/corray333/backend-labs/order/internal/service/models/outbox"
	"github.com/streadway/amqp"
)

type AuditRabbitMQRepository struct {
	client        *rabbitmq.Client
	queue         amqp.Queue
	pgClient      *postgres.Client
	outboxRepo    ioutboxrepo.IOutboxRepository
	maxRetries    int
	retryInterval time.Duration
}

func NewAuditRabbitMQRepository(
	client *rabbitmq.Client,
	pgClient *postgres.Client,
	outboxRepo ioutboxrepo.IOutboxRepository,
) *AuditRabbitMQRepository {
	queue, err := client.DeclareQueue(rabbitmq.DeclareQueueConfig{
		Name:       "oms.order.created",
		Durable:    false,
		Exclusive:  false,
		AutoDelete: false,
	})
	if err != nil {
		panic(err)
	}

	return &AuditRabbitMQRepository{
		client:        client,
		queue:         queue,
		pgClient:      pgClient,
		outboxRepo:    outboxRepo,
		maxRetries:    5,
		retryInterval: 30 * time.Second, // 30 seconds between retries
	}
}

func (r *AuditRabbitMQRepository) LogBatchInsert(ctx context.Context, orders []order.Order) error {
	now := time.Now()

	for _, ord := range orders {
		orderData, err := json.Marshal(ord)
		if err != nil {
			slog.Error("Failed to marshal order", "order_id", ord.ID, "error", err)

			return fmt.Errorf("failed to marshal order: %w", err)
		}

		// Try to publish to RabbitMQ synchronously
		err = r.client.Channel().Publish(
			"",
			r.queue.Name,
			false,
			false,
			amqp.Publishing{
				ContentType: "application/json",
				Body:        orderData,
			},
		)

		// If publish fails, save to outbox
		if err != nil {
			slog.Warn(
				"Failed to publish to RabbitMQ, saving to outbox",
				"order_id",
				ord.ID,
				"error",
				err,
			)

			outboxMsg := outbox.OutboxMessage{
				QueueName:    r.queue.Name,
				ExchangeName: "",
				RoutingKey:   r.queue.Name,
				Payload:      orderData,
				ContentType:  "application/json",
				RetryCount:   0,
				MaxRetries:   r.maxRetries,
				LastError:    err.Error(),
				CreatedAt:    now,
				UpdatedAt:    now,
				NextRetryAt:  now.Add(r.retryInterval),
			}

			if err := r.outboxRepo.Insert(ctx, outboxMsg); err != nil {
				slog.Error("Failed to insert message to outbox", "order_id", ord.ID, "error", err)

				return fmt.Errorf("failed to insert message to outbox: %w", err)
			}

			slog.Info("Message saved to outbox", "order_id", ord.ID)
		} else {
			slog.Info("Message published to RabbitMQ", "order_id", ord.ID)
		}
	}

	return nil
}

// SaveAuditLogs saves audit log entries to the PostgreSQL database using squirrel bulk insert.
func (r *AuditRabbitMQRepository) SaveAuditLogs(
	ctx context.Context,
	auditLogs []auditlog.AuditLogOrder,
) error {
	if len(auditLogs) == 0 {
		return nil
	}

	builder := sq.Insert("audit_log_order").
		Columns(
			"order_id",
			"order_item_id",
			"customer_id",
			"order_status",
			"created_at",
			"updated_at",
		).
		PlaceholderFormat(sq.Dollar)

	for _, auditLog := range auditLogs {
		builder = builder.Values(
			auditLog.OrderID,
			auditLog.OrderItemID,
			auditLog.CustomerID,
			auditLog.OrderStatus,
			auditLog.CreatedAt,
			auditLog.UpdatedAt,
		)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build audit logs insert query: %w", err)
	}

	_, err = r.pgClient.DB().ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to bulk insert audit logs: %w", err)
	}

	return nil
}
