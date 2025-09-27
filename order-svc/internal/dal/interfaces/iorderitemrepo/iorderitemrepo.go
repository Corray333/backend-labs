package iorderitem

import (
	"context"

	"github.com/corray333/backend-labs/order/internal/service/models/orderitem"
)

// IOrderItemRepository is an interface for order item postgres repository.
type IOrderItemRepository interface {
	BulkInsert(ctx context.Context, orderItems []orderitem.OrderItem) ([]orderitem.OrderItem, error)
	Query(
		ctx context.Context,
		filter *orderitem.QueryOrderItemsModel,
	) ([]orderitem.OrderItem, error)
}
