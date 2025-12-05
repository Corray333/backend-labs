package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/corray333/backend-labs/order/internal/dal/interfaces/ioutboxrepo"
	"github.com/corray333/backend-labs/order/internal/dal/postgres"
	"github.com/corray333/backend-labs/order/internal/dal/rabbitmq"
	"github.com/corray333/backend-labs/order/internal/service/models/order"
	"github.com/corray333/backend-labs/order/internal/service/models/outbox"
	"github.com/spf13/viper"
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
	queueName := viper.GetString("rabbitmq.queue.name")
	if queueName == "" {
		queueName = "oms.order.created"
	}

	queue, err := client.DeclareQueue(rabbitmq.DeclareQueueConfig{
		Name:       queueName,
		Durable:    viper.GetBool("rabbitmq.queue.durable"),
		Exclusive:  viper.GetBool("rabbitmq.queue.exclusive"),
		AutoDelete: viper.GetBool("rabbitmq.queue.auto_delete"),
		NoWait:     viper.GetBool("rabbitmq.queue.no_wait"),
	})
	if err != nil {
		panic(err)
	}

	maxRetries := viper.GetInt("rabbitmq.outbox.max_retries")
	if maxRetries == 0 {
		maxRetries = 5
	}

	retryIntervalSeconds := viper.GetInt("rabbitmq.outbox.retry_interval_seconds")
	if retryIntervalSeconds == 0 {
		retryIntervalSeconds = 30
	}

	return &AuditRabbitMQRepository{
		client:        client,
		queue:         queue,
		pgClient:      pgClient,
		outboxRepo:    outboxRepo,
		maxRetries:    maxRetries,
		retryInterval: time.Duration(retryIntervalSeconds) * time.Second,
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

