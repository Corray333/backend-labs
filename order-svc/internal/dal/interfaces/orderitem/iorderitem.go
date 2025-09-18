package iorderitem

import (
	"context"

	"github.com/corray333/backend-labs/order/internal/service/models/orderitem"
)

// PostgresRepository is an interface for order item postgres repository.
type PostgresRepository interface {
	BulkInsert(ctx context.Context, orderItems []orderitem.OrderItem) ([]orderitem.OrderItem, error)
	Query(
		ctx context.Context,
		filter *orderitem.QueryOrderItemsModel,
	) ([]orderitem.OrderItem, error)
}
