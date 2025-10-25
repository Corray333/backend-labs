package iauditrepo

import (
	"context"

	"github.com/corray333/backend-labs/consumer/internal/service/models"
)

// IAuditRepository is interface for audit repository.
type IAuditRepository interface {
	SaveAuditLogs(ctx context.Context, auditLogs []models.AuditLogOrder) error
}
