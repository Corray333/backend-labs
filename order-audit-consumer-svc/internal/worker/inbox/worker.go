package inbox

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"time"

	"github.com/corray333/backend-labs/consumer/internal/dal/interfaces/iinboxrepo"
	"github.com/corray333/backend-labs/consumer/internal/service/models"
	"github.com/corray333/backend-labs/consumer/internal/service/models/order"
)

// service represents the service layer interface.
type service interface {
	ProcessAuditLog(ctx context.Context, auditLog models.AuditLogOrder) error
}

// Worker processes messages from the inbox table.
type Worker struct {
	inboxRepo    iinboxrepo.IInboxRepository
	service      service
	pollInterval time.Duration
	batchSize    int
	stopCh       chan struct{}
}

// NewWorker creates a new inbox worker.
func NewWorker(
	inboxRepo iinboxrepo.IInboxRepository,
	service service,
	pollInterval time.Duration,
	batchSize int,
) *Worker {
	return &Worker{
		inboxRepo:    inboxRepo,
		service:      service,
		pollInterval: pollInterval,
		batchSize:    batchSize,
		stopCh:       make(chan struct{}),
	}
}

// Start begins processing messages from the inbox.
func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	slog.Info("Inbox worker started", "poll_interval", w.pollInterval, "batch_size", w.batchSize)

	for {
		select {
		case <-ctx.Done():
			slog.Info("Inbox worker shutting down")
			return
		case <-w.stopCh:
			slog.Info("Inbox worker stopped")
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

// processMessages retrieves and processes pending messages from the inbox.
func (w *Worker) processMessages(ctx context.Context) {
	messages, err := w.inboxRepo.GetPendingMessages(ctx, w.batchSize)
	if err != nil {
		slog.Error("Failed to get pending messages from inbox", "error", err)
		return
	}

	if len(messages) == 0 {
		return
	}

	slog.Info("Processing inbox messages", "count", len(messages))

	for _, msg := range messages {
		// Unmarshal the order
		var ord order.Order
		if err := json.Unmarshal(msg.Payload, &ord); err != nil {
			slog.Error("Failed to unmarshal order from inbox", "error", err, "inbox_id", msg.ID)

			// If unmarshal fails, increment retry or delete if max retries reached
			newRetryCount := msg.RetryCount + 1
			if newRetryCount >= msg.MaxRetries {
				slog.Warn("Max retries reached for malformed message, deleting",
					"inbox_id", msg.ID,
					"message_id", msg.MessageID,
				)
				if err := w.inboxRepo.Delete(ctx, msg.ID); err != nil {
					slog.Error("Failed to delete message from inbox", "inbox_id", msg.ID, "error", err)
				}
			} else {
				backoffSeconds := math.Pow(2, float64(newRetryCount)) * 30
				nextRetryAt := time.Now().Add(time.Duration(backoffSeconds) * time.Second)
				if err := w.inboxRepo.UpdateRetry(ctx, msg.ID, newRetryCount, err.Error(), nextRetryAt); err != nil {
					slog.Error("Failed to update retry information", "inbox_id", msg.ID, "error", err)
				}
			}
			continue
		}

		// Convert order to audit logs
		auditLogs := w.convertOrderToAuditLogs(ord)

		// Try to process each audit log
		var processingErr error
		for _, auditLog := range auditLogs {
			if err := w.service.ProcessAuditLog(ctx, auditLog); err != nil {
				processingErr = err
				slog.Error("Failed to process audit log from inbox", "error", err, "inbox_id", msg.ID, "order_id", ord.ID)
				break
			}
		}

		if processingErr != nil {
			// Update retry count and schedule next retry with exponential backoff
			newRetryCount := msg.RetryCount + 1
			backoffSeconds := math.Pow(2, float64(newRetryCount)) * 30 // 30s, 60s, 120s, 240s, etc.
			nextRetryAt := time.Now().Add(time.Duration(backoffSeconds) * time.Second)

			slog.Warn("Failed to process message from inbox, will retry",
				"inbox_id", msg.ID,
				"retry_count", newRetryCount,
				"next_retry", nextRetryAt,
				"error", processingErr,
			)

			if err := w.inboxRepo.UpdateRetry(ctx, msg.ID, newRetryCount, processingErr.Error(), nextRetryAt); err != nil {
				slog.Error("Failed to update retry information", "inbox_id", msg.ID, "error", err)
			}
		} else {
			// Successfully processed, delete from inbox
			if err := w.inboxRepo.Delete(ctx, msg.ID); err != nil {
				slog.Error("Failed to delete message from inbox after successful processing",
					"inbox_id", msg.ID,
					"error", err,
				)
			} else {
				slog.Info("Message successfully processed and removed from inbox",
					"inbox_id", msg.ID,
					"message_id", msg.MessageID,
					"order_id", ord.ID,
				)
			}
		}
	}
}

// convertOrderToAuditLogs converts an order to audit log entries.
func (w *Worker) convertOrderToAuditLogs(ord order.Order) []models.AuditLogOrder {
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
