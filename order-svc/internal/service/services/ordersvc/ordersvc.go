package ordersvc

import (
	"context"
	"log/slog"
	"time"

	"github.com/corray333/backend-labs/order/internal/dal/interfaces/iauditrepo"
	iorderitem "github.com/corray333/backend-labs/order/internal/dal/interfaces/iorderitemrepo"
	iorder "github.com/corray333/backend-labs/order/internal/dal/interfaces/iorderrepo"
	"github.com/corray333/backend-labs/order/internal/dal/postgres"
	"github.com/corray333/backend-labs/order/internal/dal/uow"
	"github.com/corray333/backend-labs/order/internal/service/models/auditlog"
	"github.com/corray333/backend-labs/order/internal/service/models/order"
	"github.com/corray333/backend-labs/order/internal/service/models/orderitem"
	"go.opentelemetry.io/otel"
)

// OrderService is a service for managing orders.
type OrderService struct {
	pgClient *postgres.Client
	auditor  iauditrepo.IAuditorRepository
}

func (s *OrderService) newUOW() unitOfWork {
	return uow.NewUnitOfWork(s.pgClient)
}

type unitOfWork interface {
	Begin(ctx context.Context) error
	Commit() error
	Rollback() error

	OrderRepository() iorder.IOrderRepository
	OrderItemRepository() iorderitem.IOrderItemRepository
}

// option is a function that configures the OrderService.
type option func(*OrderService)

// MustNewOrderService creates a new OrderService.
func MustNewOrderService(opts ...option) *OrderService {
	s := &OrderService{}
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// WithAuditor sets the Auditor for the OrderService.
//
//goland:noinspection GoExportedFuncWithUnexportedType
func WithAuditor(auditor iauditrepo.IAuditorRepository) option {
	return func(s *OrderService) {
		s.auditor = auditor
	}
}

// WithPostgresClient sets the Postgres client for the OrderService.
//
//goland:noinspection GoExportedFuncWithUnexportedType
func WithPostgresClient(pgClient *postgres.Client) option {
	return func(s *OrderService) {
		s.pgClient = pgClient
	}
}

// BatchInsert creates multiple orders with their items in a transaction.
func (s *OrderService) BatchInsert(
	ctx context.Context,
	orders []order.Order,
) ([]order.Order, error) {
	ctx, span := otel.Tracer("service").Start(ctx, "Service.CreateOrders")
	defer span.End()

	now := time.Now()

	work := s.newUOW()

	err := work.Begin(ctx)
	if err != nil {
		return nil, err
	}
	for i := range orders {
		orders[i].CreatedAt = now
		orders[i].UpdatedAt = now
	}

	orders, err = work.OrderRepository().BulkInsert(ctx, orders)
	if err != nil {
		return nil, err
	}

	orderItems := make([]orderitem.OrderItem, 0)
	for _, o := range orders {
		for _, item := range o.OrderItems {
			item.OrderID = o.ID
			orderItems = append(orderItems, item)
		}
	}
	orderItems, err = work.OrderItemRepository().BulkInsert(ctx, orderItems)
	if err != nil {
		return nil, err
	}

	for i := range orders {
		orders[i].OrderItems = orderItems[i*len(orders[i].OrderItems) : (i+1)*len(orders[i].OrderItems)]
	}

	// Try to send audit logs to RabbitMQ synchronously before committing
	err = s.auditor.LogBatchInsert(ctx, orders)
	if err != nil {
		// Rollback transaction if audit logging fails
		if rbErr := work.Rollback(); rbErr != nil {
			slog.Error("Failed to rollback transaction", "error", rbErr)
		}

		return nil, err
	}

	// Commit transaction only if audit logging succeeded
	err = work.Commit()
	if err != nil {
		return nil, err
	}

	return orders, nil
}

// GetOrders retrieves orders with optional iorderrepo items based on filter.
func (s *OrderService) GetOrders(
	ctx context.Context,
	model orderitem.QueryOrderItemsModel,
) ([]order.Order, error) {
	ctx, span := otel.Tracer("service").Start(ctx, "Service.GetOrders")
	defer span.End()

	orderQuery := &order.QueryOrdersModel{
		Ids:         model.Ids,
		CustomerIds: model.CustomerIds,
		Limit:       model.PageSize,
		Offset:      (model.Page - 1) * model.PageSize,
	}

	work := s.newUOW()

	orders, err := work.OrderRepository().Query(ctx, orderQuery)
	if err != nil {
		return nil, err
	}

	if len(orders) == 0 {
		return []order.Order{}, nil
	}

	orderItemQuery := &orderitem.QueryOrderItemsModel{}
	for _, o := range orders {
		orderItemQuery.OrderIds = append(orderItemQuery.OrderIds, o.ID)
	}
	orderItems, err := work.OrderItemRepository().Query(ctx, orderItemQuery)
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

// SaveAuditLogs saves audit log entries to the database.
func (s *OrderService) SaveAuditLogs(
	ctx context.Context,
	auditLogs []auditlog.AuditLogOrder,
) ([]auditlog.AuditLogOrder, error) {
	now := time.Now()

	// Set created and updated timestamps
	for i := range auditLogs {
		if auditLogs[i].CreatedAt.IsZero() {
			auditLogs[i].CreatedAt = now
		}
		if auditLogs[i].UpdatedAt.IsZero() {
			auditLogs[i].UpdatedAt = now
		}
	}

	// Save audit logs using the auditor repository
	err := s.auditor.SaveAuditLogs(ctx, auditLogs)
	if err != nil {
		return nil, err
	}

	slog.Info("Audit logs saved successfully", "count", len(auditLogs))

	return auditLogs, nil
}
