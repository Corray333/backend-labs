package consumer

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/corray333/backend-labs/consumer/internal/rabbitmq"
	"github.com/corray333/backend-labs/consumer/internal/service/models"
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
	client  *rabbitmq.Client
	service service
	queue   amqp.Queue
	stop    chan struct{}
	done    chan struct{}
}

// NewConsumer creates a new Consumer.
func NewConsumer(client *rabbitmq.Client, service service) *Consumer {
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

	return &Consumer{
		client:  client,
		service: service,
		queue:   queue,
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
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
		// Reject the message without requeuing
		if err := msg.Nack(false, false); err != nil {
			slog.Error("Failed to nack message", "error", err)
		}

		return err
	}

	// Convert order to audit logs
	auditLogs := c.convertOrderToAuditLogs(ord)

	// Process each audit log
	for _, auditLog := range auditLogs {
		if err := c.service.ProcessAuditLog(ctx, auditLog); err != nil {
			slog.Error("Failed to process audit log", "error", err, "order_id", ord.ID)
			// Requeue the message for retry
			if err := msg.Nack(false, true); err != nil {
				slog.Error("Failed to nack message", "error", err)
			}

			return err
		}
	}

	// Acknowledge the message
	if err := msg.Ack(false); err != nil {
		slog.Error("Failed to ack message", "error", err)

		return err
	}

	slog.Info("Message processed successfully", "order_id", ord.ID)

	return nil
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
