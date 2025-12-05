package postgres

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/corray333/backend-labs/consumer/internal/dal/postgres"
	"github.com/corray333/backend-labs/consumer/internal/service/models"
)

// AuditRepository implements the audit repository for PostgreSQL.
type AuditRepository struct {
	pgClient *postgres.Client
}

// NewAuditRepository creates a new audit repository.
func NewAuditRepository(pgClient *postgres.Client) *AuditRepository {
	return &AuditRepository{
		pgClient: pgClient,
	}
}

// SaveAuditLogs saves audit log entries to the PostgreSQL database using squirrel bulk insert.
func (r *AuditRepository) SaveAuditLogs(
	ctx context.Context,
	auditLogs []models.AuditLogOrder,
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

	_, err = r.pgClient.Pool().Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to bulk insert audit logs: %w", err)
	}

	return nil
}
