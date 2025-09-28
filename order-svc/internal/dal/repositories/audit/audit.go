package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/corray333/backend-labs/order/internal/dal/postgres"
	"github.com/corray333/backend-labs/order/internal/dal/rabbitmq"
	"github.com/corray333/backend-labs/order/internal/service/models/auditlog"
	"github.com/corray333/backend-labs/order/internal/service/models/order"
	"github.com/lib/pq"
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
		Name:       "oms.iorderrepo.created",
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
	auditCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	g, ctx := errgroup.WithContext(auditCtx)
	g.SetLimit(3)

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

// SaveAuditLogs saves audit log entries to the PostgreSQL database using bulk insert with unnest().
func (r *AuditRabbitMQRepository) SaveAuditLogs(
	ctx context.Context,
	auditLogs []auditlog.AuditLogOrder,
) error {
	if len(auditLogs) == 0 {
		return nil
	}

	query := `
		INSERT INTO audit_log_order (
			order_id,
			order_item_id,
			customer_id,
			order_status,
			created_at,
			updated_at
		)
		SELECT
			order_id,
			order_item_id,
			customer_id,
			order_status,
			created_at,
			updated_at
		FROM unnest($1::bigint[], $2::bigint[], $3::bigint[], $4::text[], $5::timestamp[], $6::timestamp[])
		AS t(order_id, order_item_id, customer_id, order_status, created_at, updated_at)
	`

	// Prepare arrays for bulk insert
	orderIds := make([]int64, len(auditLogs))
	orderItemIds := make([]int64, len(auditLogs))
	customerIds := make([]int64, len(auditLogs))
	orderStatuses := make([]string, len(auditLogs))
	createdAts := make([]time.Time, len(auditLogs))
	updatedAts := make([]time.Time, len(auditLogs))

	for i, auditLog := range auditLogs {
		orderIds[i] = auditLog.OrderID
		orderItemIds[i] = auditLog.OrderItemID
		customerIds[i] = auditLog.CustomerID
		orderStatuses[i] = auditLog.OrderStatus
		createdAts[i] = auditLog.CreatedAt
		updatedAts[i] = auditLog.UpdatedAt
	}

	_, err := r.pgClient.DB().ExecContext(ctx, query,
		pq.Array(orderIds),
		pq.Array(orderItemIds),
		pq.Array(customerIds),
		pq.Array(orderStatuses),
		pq.Array(createdAts),
		pq.Array(updatedAts),
	)
	if err != nil {
		return fmt.Errorf("failed to bulk insert audit logs: %w", err)
	}

	return nil
}
