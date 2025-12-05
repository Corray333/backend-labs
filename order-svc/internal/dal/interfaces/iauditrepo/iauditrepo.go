package iauditrepo

import (
	"context"

	"github.com/corray333/backend-labs/order/internal/service/models/order"
)

// IAuditorRepository is interface for auditor repository.
type IAuditorRepository interface {
	LogBatchInsert(ctx context.Context, orders []order.Order) error
}
