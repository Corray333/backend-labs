package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/corray333/backend-labs/order/internal/dal/postgres"
	"github.com/corray333/backend-labs/order/internal/dal/rabbitmq"
	"github.com/corray333/backend-labs/order/internal/service/models/auditlog"
	"github.com/corray333/backend-labs/order/internal/service/models/order"
	"github.com/streadway/amqp"
	"golang.org/x/sync/errgroup"
)

type AuditRabbitMQRepository struct {
	client   *rabbitmq.Client
	queue    amqp.Queue
	pgClient *postgres.Client
}

func NewAuditRabbitMQRepository(
	client *rabbitmq.Client,
	pgClient *postgres.Client,
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
		client:   client,
		queue:    queue,
		pgClient: pgClient,
	}
}

func (r *AuditRabbitMQRepository) LogBatchInsert(ctx context.Context, orders []order.Order) error {
	auditCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	g, ctx := errgroup.WithContext(auditCtx)
	g.SetLimit(5)

	for _, ord := range orders {
		g.Go(func() error {
			orderData, err := json.Marshal(ord)
			if err != nil {
				return err
			}

			return r.client.Channel().Publish(
				"",
				r.queue.Name,
				false,
				false,
				amqp.Publishing{
					ContentType: "application/json",
					Body:        orderData,
				},
			)
		})
	}

	return g.Wait()
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
