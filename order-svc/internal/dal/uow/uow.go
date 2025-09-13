package uow

import (
	"context"

	iorder "github.com/corray333/backend-labs/order/internal/dal/interfaces/order"
	iorderitem "github.com/corray333/backend-labs/order/internal/dal/interfaces/orderitem"
	"github.com/corray333/backend-labs/order/internal/dal/postgres"
	orderrepo "github.com/corray333/backend-labs/order/internal/dal/repositories/order/postgres"
	orderitemrepo "github.com/corray333/backend-labs/order/internal/dal/repositories/orderitem/postgres"

	"github.com/jmoiron/sqlx"
)

type unitOfWork struct {
	db            *sqlx.DB
	tx            *sqlx.Tx
	orderRepo     iorder.IOrderPostgresRepository
	orderItemRepo iorderitem.IOrderItemPostgresRepository
}

func (u *unitOfWork) OrderRepository() iorder.IOrderPostgresRepository {
	return u.orderRepo
}

func (u *unitOfWork) OrderItemRepository() iorderitem.IOrderItemPostgresRepository {
	return u.orderItemRepo
}

func NewUnitOfWork(db *postgres.PostgresClient) *unitOfWork {
	return &unitOfWork{
		db:            db.DB(),
		orderRepo:     orderrepo.NewPostgresOrderRepository(db.DB()),
		orderItemRepo: orderitemrepo.NewPostgresOrderItemRepository(db.DB()),
	}
}

func (u *unitOfWork) Begin(ctx context.Context) error {
	tx, err := u.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	u.tx = tx
	// Создаем репозитории с транзакцией
	u.orderRepo = orderrepo.NewPostgresOrderRepository(tx)
	u.orderItemRepo = orderitemrepo.NewPostgresOrderItemRepository(tx)

	return nil
}

func (u *unitOfWork) Commit() error {
	if u.tx == nil {
		return nil
	}
	return u.tx.Commit()
}

func (u *unitOfWork) Rollback() error {
	if u.tx == nil {
		return nil
	}
	return u.tx.Rollback()
}
