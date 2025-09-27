package uow

import (
	"context"

	iorderitem "github.com/corray333/backend-labs/order/internal/dal/interfaces/iorderitemrepo"
	iorder "github.com/corray333/backend-labs/order/internal/dal/interfaces/iorderrepo"
	"github.com/corray333/backend-labs/order/internal/dal/postgres"
	orderrepo "github.com/corray333/backend-labs/order/internal/dal/repositories/order/postgres"
	orderitemrepo "github.com/corray333/backend-labs/order/internal/dal/repositories/orderitem/postgres"
	"github.com/jmoiron/sqlx"
)

type unitOfWork struct {
	db            *sqlx.DB
	tx            *sqlx.Tx
	orderRepo     iorder.IOrderRepository
	orderItemRepo iorderitem.IOrderItemRepository
}

// OrderRepository returns iorderrepo repository.
func (u *unitOfWork) OrderRepository() iorder.IOrderRepository {
	return u.orderRepo
}

// OrderItemRepository returns iorderrepo item repository.
func (u *unitOfWork) OrderItemRepository() iorderitem.IOrderItemRepository {
	return u.orderItemRepo
}

// NewUnitOfWork creates new unit of work.
//
//goland:noinspection GoExportedFuncWithUnexportedType
func NewUnitOfWork(db *postgres.Client) *unitOfWork {
	return &unitOfWork{
		db:            db.DB(),
		orderRepo:     orderrepo.NewPostgresOrderRepository(db.DB()),
		orderItemRepo: orderitemrepo.NewPostgresOrderItemRepository(db.DB()),
	}
}

// Begin starts a new transaction.
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

// Commit commits the transaction.
func (u *unitOfWork) Commit() error {
	if u.tx == nil {
		return nil
	}

	return u.tx.Commit()
}

// Rollback rolls back the transaction.
func (u *unitOfWork) Rollback() error {
	if u.tx == nil {
		return nil
	}

	return u.tx.Rollback()
}
