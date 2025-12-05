package consumer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/corray333/backend-labs/consumer/internal/dal/interfaces/iinboxrepo"
	"github.com/corray333/backend-labs/consumer/internal/rabbitmq"
	"github.com/corray333/backend-labs/consumer/internal/service/models"
	"github.com/corray333/backend-labs/consumer/internal/service/models/inbox"
	"github.com/corray333/backend-labs/consumer/internal/service/models/order"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
	"go.opentelemetry.io/otel"
	"golang.org/x/sync/errgroup"
)

// service represents the service layer interface.
type service interface {
	ProcessAuditLog(ctx context.Context, auditLog models.AuditLogOrder) error
}

// Consumer represents the RabbitMQ consumer transport.
type Consumer struct {
	client     *rabbitmq.Client
	service    service
	inboxRepo  iinboxrepo.IInboxRepository
	queue      amqp.Queue
	stop       chan struct{}
	done       chan struct{}
	maxRetries int
}

// NewConsumer creates a new Consumer.
func NewConsumer(
	client *rabbitmq.Client,
	service service,
	inboxRepo iinboxrepo.IInboxRepository,
) *Consumer {
	queueName := viper.GetString("rabbitmq.queue")
	if queueName == "" {
		panic("rabbitmq.queue is not set in config")
	}

	queue, err := client.DeclareQueue(rabbitmq.DeclareQueueConfig{
		Name:       queueName,
		Durable:    false,
		AutoDelete: false,
		Exclusive:  false,
		NoWait:     false,
	})
	if err != nil {
		panic(err)
	}

	maxRetries := viper.GetInt("inbox.max_retries")
	if maxRetries == 0 {
		maxRetries = 5
	}

	return &Consumer{
		client:     client,
		service:    service,
		inboxRepo:  inboxRepo,
		queue:      queue,
		stop:       make(chan struct{}),
		done:       make(chan struct{}),
		maxRetries: maxRetries,
	}
}

// Run starts consuming messages from RabbitMQ.
func (c *Consumer) Run(ctx context.Context) error {
	consumerTag := viper.GetString("rabbitmq.consumer_tag")
	if consumerTag == "" {
		consumerTag = "consumer-svc"
	}

	msgs, err := c.client.Consume(rabbitmq.ConsumeConfig{
		Queue:     c.queue.Name,
		Consumer:  consumerTag,
		AutoAck:   viper.GetBool("rabbitmq.auto_ack"),
		Exclusive: viper.GetBool("rabbitmq.exclusive"),
		NoLocal:   viper.GetBool("rabbitmq.no_local"),
		NoWait:    viper.GetBool("rabbitmq.no_wait"),
	})
	if err != nil {
		return err
	}

	slog.Info("Consumer started", "queue", c.queue.Name, "consumer_tag", consumerTag)

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(50)

	go func() {
		for {
			select {
			case <-c.stop:
				slog.Info("Stopping consumer")
				close(c.done)

				return
			case msg, ok := <-msgs:
				if !ok {
					slog.Info("Message channel closed")
					close(c.done)

					return
				}

				g.Go(func() error {
					return c.processMessage(gctx, msg)
				})
			}
		}
	}()

	<-c.done
	if err := g.Wait(); err != nil {
		slog.Error("Error processing messages", "error", err)
	}

	return nil
}

// processMessage processes a single message from RabbitMQ.
func (c *Consumer) processMessage(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := otel.Tracer("consumer").Start(ctx, "Consumer.processMessage")
	defer span.End()

	slog.Info("Received message", "delivery_tag", msg.DeliveryTag)

	// Unmarshal the order
	var ord order.Order
	if err := json.Unmarshal(msg.Body, &ord); err != nil {
		slog.Error("Failed to unmarshal order", "error", err)
		// Reject the message without requeuing for malformed messages
		if err := msg.Nack(false, false); err != nil {
			slog.Error("Failed to nack message", "error", err)
		}

		return err
	}

	// Convert order to audit logs
	auditLogs := c.convertOrderToAuditLogs(ord)

	// Try to process each audit log
	var processingErr error
	for _, auditLog := range auditLogs {
		if err := c.service.ProcessAuditLog(ctx, auditLog); err != nil {
			processingErr = err
			slog.Error("Failed to process audit log", "error", err, "order_id", ord.ID)
			break
		}
	}

	// If processing failed, save to inbox and acknowledge
	if processingErr != nil {
		messageID := c.generateMessageID(msg.Body)
		inboxMsg := inbox.InboxMessage{
			MessageID:   messageID,
			QueueName:   c.queue.Name,
			RoutingKey:  msg.RoutingKey,
			Payload:     msg.Body,
			ContentType: msg.ContentType,
			RetryCount:  0,
			MaxRetries:  c.maxRetries,
			LastError:   processingErr.Error(),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			NextRetryAt: time.Now().Add(30 * time.Second),
			DeliveryTag: msg.DeliveryTag,
		}

		if err := c.inboxRepo.Insert(ctx, inboxMsg); err != nil {
			slog.Error("Failed to save message to inbox", "error", err, "order_id", ord.ID)
			// Don't acknowledge - let RabbitMQ requeue
			if err := msg.Nack(false, true); err != nil {
				slog.Error("Failed to nack message", "error", err)
			}
			return fmt.Errorf("failed to save to inbox: %w", err)
		}

		slog.Info("Message saved to inbox for retry", "order_id", ord.ID, "message_id", messageID)
	}

	// Acknowledge the message (either processed successfully or saved to inbox)
	if err := msg.Ack(false); err != nil {
		slog.Error("Failed to ack message", "error", err)
		return err
	}

	if processingErr == nil {
		slog.Info("Message processed successfully", "order_id", ord.ID)
	}

	return nil
}

// generateMessageID generates a unique message ID based on message content.
func (c *Consumer) generateMessageID(payload []byte) string {
	hash := sha256.Sum256(payload)
	return hex.EncodeToString(hash[:])
}

// convertOrderToAuditLogs converts an order to audit log entries.
func (c *Consumer) convertOrderToAuditLogs(ord order.Order) []models.AuditLogOrder {
	auditLogs := make([]models.AuditLogOrder, 0, len(ord.OrderItems))

	for _, item := range ord.OrderItems {
		auditLog := models.AuditLogOrder{
			OrderID:     ord.ID,
			OrderItemID: item.ID,
			CustomerID:  ord.CustomerID,
			OrderStatus: "created",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		auditLogs = append(auditLogs, auditLog)
	}

	return auditLogs
}

// Shutdown gracefully shuts down the consumer.
func (c *Consumer) Shutdown() error {
	slog.Info("Shutting down consumer")
	close(c.stop)

	// Wait for processing to finish with timeout
	select {
	case <-c.done:
		slog.Info("Consumer stopped successfully")
	case <-time.After(10 * time.Second):
		slog.Warn("Consumer shutdown timeout")
	}

	return nil
}
