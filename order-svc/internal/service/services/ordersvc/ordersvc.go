package ordersvc

import (
	"context"
	"time"

	iorder "github.com/corray333/backend-labs/order/internal/dal/interfaces/order"
	iorderitem "github.com/corray333/backend-labs/order/internal/dal/interfaces/orderitem"
	"github.com/corray333/backend-labs/order/internal/dal/postgres"
	"github.com/corray333/backend-labs/order/internal/dal/uow"
	"github.com/corray333/backend-labs/order/internal/service/models/order"
	"github.com/corray333/backend-labs/order/internal/service/models/orderitem"
)

type OrderService struct {
	pgClient *postgres.PostgresClient
}

func (s *OrderService) newUOW(ctx context.Context) unitOfWork {
	return uow.NewUnitOfWork(s.pgClient)
}

type unitOfWork interface {
	Begin(ctx context.Context) error
	Commit() error
	Rollback() error

	OrderRepository() iorder.IOrderPostgresRepository
	OrderItemRepository() iorderitem.IOrderItemPostgresRepository
}

type option func(*OrderService)

func MustNewOrderService(opts ...option) *OrderService {
	s := &OrderService{}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func WithPostgresClient(pgClient *postgres.PostgresClient) option {
	return func(s *OrderService) {
		s.pgClient = pgClient
	}
}

// BatchInsert creates multiple orders with their items in a transaction
func (s *OrderService) BatchInsert(ctx context.Context, orders []order.Order) ([]order.Order, error) {
	now := time.Now()

	uow := s.newUOW(ctx)

	err := uow.Begin(ctx)
	if err != nil {
		return nil, err
	}
	for i := range orders {
		orders[i].CreatedAt = now
		orders[i].UpdatedAt = now
	}

	orders, err = uow.OrderRepository().BulkInsert(ctx, orders)
	if err != nil {
		return nil, err
	}

	orderItems := make([]orderitem.OrderItem, 0)
	for _, order := range orders {
		for _, item := range order.OrderItems {
			item.OrderID = order.ID
			orderItems = append(orderItems, item)
		}
	}
	orderItems, err = uow.OrderItemRepository().BulkInsert(ctx, orderItems)
	if err != nil {
		return nil, err
	}

	for i := range orders {
		orders[i].OrderItems = orderItems[i*len(orders[i].OrderItems) : (i+1)*len(orders[i].OrderItems)]
	}

	err = uow.Commit()
	if err != nil {
		return nil, err
	}

	return orders, nil
}

// GetOrders retrieves orders with optional order items based on filter
func (s *OrderService) GetOrders(ctx context.Context, model orderitem.QueryOrderItemsModel) ([]order.Order, error) {
	orderQuery := &order.QueryOrdersModel{
		Ids:         model.Ids,
		CustomerIds: model.CustomerIds,
		Limit:       model.PageSize,
		Offset:      (model.Page - 1) * model.PageSize,
	}

	uow := s.newUOW(ctx)

	orders, err := uow.OrderRepository().Query(ctx, orderQuery)
	if err != nil {
		return nil, err
	}

	if len(orders) == 0 {
		return []order.Order{}, nil
	}

	orderItemQuery := &orderitem.QueryOrderItemsModel{}
	for _, order := range orders {
		orderItemQuery.OrderIds = append(orderItemQuery.OrderIds, order.ID)
	}
	orderItems, err := uow.OrderItemRepository().Query(ctx, orderItemQuery)
	if err != nil {
		return nil, err
	}

	for i := range orders {
		for _, item := range orderItems {
			if item.OrderID == orders[i].ID {
				orders[i].OrderItems = append(orders[i].OrderItems, item)
			}
		}
	}

	return orders, nil
}
