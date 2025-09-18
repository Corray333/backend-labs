package iorder

import (
	"context"

	"github.com/corray333/backend-labs/order/internal/service/models/order"
)

// PostgresRepository is an interface for order postgres repository.
type PostgresRepository interface {
	BulkInsert(ctx context.Context, orders []order.Order) ([]order.Order, error)
	Query(ctx context.Context, filter *order.QueryOrdersModel) ([]order.Order, error)
}
