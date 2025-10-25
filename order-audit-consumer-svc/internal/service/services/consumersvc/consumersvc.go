package consumersvc

import (
	"context"
	"log/slog"

	"github.com/corray333/backend-labs/consumer/internal/dal/interfaces/iauditrepo"
	"github.com/corray333/backend-labs/consumer/internal/service/models"
	"go.opentelemetry.io/otel"
)

// ConsumerService is a service for consuming audit logs.
type ConsumerService struct {
	auditRepo iauditrepo.IAuditRepository
}

// option is a function that configures the ConsumerService.
type option func(*ConsumerService)

// MustNewConsumerService creates a new ConsumerService.
func MustNewConsumerService(opts ...option) *ConsumerService {
	s := &ConsumerService{}
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// WithAuditRepository sets the audit repository for the ConsumerService.
//
//goland:noinspection GoExportedFuncWithUnexportedType
func WithAuditRepository(auditRepo iauditrepo.IAuditRepository) option {
	return func(s *ConsumerService) {
		s.auditRepo = auditRepo
	}
}

// ProcessAuditLog processes a single audit log by sending it to the repository.
func (s *ConsumerService) ProcessAuditLog(
	ctx context.Context,
	auditLog models.AuditLogOrder,
) error {
	ctx, span := otel.Tracer("service").Start(ctx, "Service.ProcessAuditLog")
	defer span.End()

	slog.Info("Processing audit log",
		"order_id", auditLog.OrderID,
		"order_item_id", auditLog.OrderItemID,
		"customer_id", auditLog.CustomerID)

	err := s.auditRepo.SaveAuditLogs(ctx, []models.AuditLogOrder{auditLog})
	if err != nil {
		slog.Error("Failed to save audit log", "error", err)

		return err
	}

	slog.Info("Audit log processed successfully", "order_id", auditLog.OrderID)

	return nil
}
