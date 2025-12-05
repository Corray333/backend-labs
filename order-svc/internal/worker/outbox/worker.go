package outbox

import (
	"context"
	"log/slog"
	"math"
	"time"

	"github.com/corray333/backend-labs/order/internal/dal/interfaces/ioutboxrepo"
	"github.com/corray333/backend-labs/order/internal/dal/rabbitmq"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
)

// Worker processes messages from the outbox table.
type Worker struct {
	outboxRepo    ioutboxrepo.IOutboxRepository
	rabbitClient  *rabbitmq.Client
	pollInterval  time.Duration
	batchSize     int
	retryInterval time.Duration
	stopCh        chan struct{}
}

// NewWorker creates a new outbox worker.
func NewWorker(
	outboxRepo ioutboxrepo.IOutboxRepository,
	rabbitClient *rabbitmq.Client,
) *Worker {
	pollIntervalSeconds := viper.GetInt("rabbitmq.outbox.poll_interval_seconds")
	if pollIntervalSeconds == 0 {
		pollIntervalSeconds = 10
	}

	batchSize := viper.GetInt("rabbitmq.outbox.batch_size")
	if batchSize == 0 {
		batchSize = 100
	}

	retryIntervalSeconds := viper.GetInt("rabbitmq.outbox.retry_interval_seconds")
	if retryIntervalSeconds == 0 {
		retryIntervalSeconds = 30
	}

	return &Worker{
		outboxRepo:    outboxRepo,
		rabbitClient:  rabbitClient,
		pollInterval:  time.Duration(pollIntervalSeconds) * time.Second,
		batchSize:     batchSize,
		retryInterval: time.Duration(retryIntervalSeconds) * time.Second,
		stopCh:        make(chan struct{}),
	}
}

// Start begins processing messages from the outbox.
func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	slog.Info("Outbox worker started", "poll_interval", w.pollInterval, "batch_size", w.batchSize)

	for {
		select {
		case <-ctx.Done():
			slog.Info("Outbox worker shutting down")

			return
		case <-w.stopCh:
			slog.Info("Outbox worker stopped")

			return
		case <-ticker.C:
			w.processMessages(ctx)
		}
	}
}

// Stop stops the worker.
func (w *Worker) Stop() {
	close(w.stopCh)
}

// processMessages retrieves and processes pending messages from the outbox.
func (w *Worker) processMessages(ctx context.Context) {
	messages, err := w.outboxRepo.GetPendingMessages(ctx, w.batchSize)
	if err != nil {
		slog.Error("Failed to get pending messages from outbox", "error", err)

		return
	}

	if len(messages) == 0 {
		return
	}

	slog.Info("Processing outbox messages", "count", len(messages))

	for _, msg := range messages {
		err := w.rabbitClient.Channel().Publish(
			msg.ExchangeName,
			msg.RoutingKey,
			false,
			false,
			amqp.Publishing{
				ContentType: msg.ContentType,
				Body:        msg.Payload,
			},
		)

		if err != nil {
			// Update retry count and schedule next retry with exponential backoff
			newRetryCount := msg.RetryCount + 1
			backoffSeconds := math.Pow(2, float64(newRetryCount)) * 30 // 30s, 60s, 120s, 240s, etc.
			nextRetryAt := time.Now().Add(time.Duration(backoffSeconds) * time.Second)

			slog.Warn("Failed to publish message from outbox, will retry",
				"outbox_id", msg.ID,
				"retry_count", newRetryCount,
				"next_retry", nextRetryAt,
				"error", err,
			)

			if err := w.outboxRepo.UpdateRetry(ctx, msg.ID, newRetryCount, err.Error(), nextRetryAt); err != nil {
				slog.Error("Failed to update retry information", "outbox_id", msg.ID, "error", err)
			}
		} else {
			// Successfully published, delete from outbox
			if err := w.outboxRepo.Delete(ctx, msg.ID); err != nil {
				slog.Error("Failed to delete message from outbox after successful publish",
					"outbox_id", msg.ID,
					"error", err,
				)
			} else {
				slog.Info("Message successfully published and removed from outbox", "outbox_id", msg.ID)
			}
		}
	}
}
