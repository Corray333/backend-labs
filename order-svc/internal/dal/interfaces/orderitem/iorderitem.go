package iorderitem

import (
	"context"

	"github.com/corray333/backend-labs/order/internal/service/models/orderitem"
)

type IOrderItemPostgresRepository interface {
	BulkInsert(ctx context.Context, orderItems []orderitem.OrderItem) ([]orderitem.OrderItem, error)
	Query(ctx context.Context, filter *orderitem.QueryOrderItemsModel) ([]orderitem.OrderItem, error)
}
