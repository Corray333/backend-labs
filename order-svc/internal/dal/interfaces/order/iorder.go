package iorder

import (
	"context"

	"github.com/corray333/backend-labs/order/internal/service/models/order"
)

type IOrderPostgresRepository interface {
	BulkInsert(ctx context.Context, orders []order.Order) ([]order.Order, error)
	Query(ctx context.Context, filter *order.QueryOrdersModel) ([]order.Order, error)
}
