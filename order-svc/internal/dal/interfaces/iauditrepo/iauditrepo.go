package iauditrepo

import (
	"context"

	"github.com/corray333/backend-labs/order/internal/service/models/auditlog"
	"github.com/corray333/backend-labs/order/internal/service/models/order"
)

// IAuditorRepository is interface for auditor repository.
type IAuditorRepository interface {
	LogBatchInsert(ctx context.Context, orders []order.Order) error
	SaveAuditLogs(ctx context.Context, auditLogs []auditlog.AuditLogOrder) error
}
