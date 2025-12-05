package postgresrepo

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/corray333/backend-labs/order/internal/service/models/currency"
	"github.com/corray333/backend-labs/order/internal/service/models/order"
	"github.com/corray333/backend-labs/order/internal/service/models/orderitem"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"go.opentelemetry.io/otel"
)

// OrderDal represents iorderrepo data access layer model.
type OrderDal struct {
	Id                 int64     `db:"id"`
	CustomerId         int64     `db:"customer_id"`
	DeliveryAddress    string    `db:"delivery_address"`
	TotalPriceCents    int64     `db:"total_price_cents"`
	TotalPriceCurrency string    `db:"total_price_currency"`
	CreatedAt          time.Time `db:"created_at"`
	UpdatedAt          time.Time `db:"updated_at"`
}

// ToModel converts OrderDal to service layer Order model.
func (o *OrderDal) ToModel() (*order.Order, error) {
	cur, err := currency.ParseCurrency(o.TotalPriceCurrency)
	if err != nil {
		return nil, err
	}

	return &order.Order{
		ID:                 o.Id,
		CustomerID:         o.CustomerId,
		DeliveryAddress:    o.DeliveryAddress,
		TotalPriceCents:    o.TotalPriceCents,
		TotalPriceCurrency: cur,
		CreatedAt:          o.CreatedAt,
		UpdatedAt:          o.UpdatedAt,
		OrderItems:         []orderitem.OrderItem{}, // Will be populated separately
	}, nil
}

// OrderDalFromModel converts service layer Order model to OrderDal.
func OrderDalFromModel(o *order.Order) *OrderDal {
	return &OrderDal{
		Id:                 o.ID,
		CustomerId:         o.CustomerID,
		DeliveryAddress:    o.DeliveryAddress,
		TotalPriceCents:    o.TotalPriceCents,
		TotalPriceCurrency: o.TotalPriceCurrency.String(),
		CreatedAt:          o.CreatedAt,
		UpdatedAt:          o.UpdatedAt,
	}
}

// PostgresOrderRepository represents a Postgres iorderrepo repository.
type PostgresOrderRepository struct {
	conn GenericConn
	sb   sq.StatementBuilderType
}

// GenericConn is an interface that works with both pgxpool.Pool and pgx.Tx
type GenericConn interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

// NewPostgresOrderRepository creates a new Postgres iorderrepo repository.
func NewPostgresOrderRepository(conn GenericConn) *PostgresOrderRepository {
	return &PostgresOrderRepository{
		conn: conn,
		sb:   sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

// BulkInsert inserts multiple orders and returns the inserted orders with IDs.
// Uses PostgreSQL composite type v1_order for optimization.
func (r *PostgresOrderRepository) BulkInsert(
	ctx context.Context,
	orders []order.Order,
) ([]order.Order, error) {
	ctx, span := otel.Tracer("dal").Start(ctx, "DAL.CreateOrders")
	defer span.End()

	if len(orders) == 0 {
		return []order.Order{}, nil
	}

	compositeRecords := make([][]interface{}, len(orders))
	for i, o := range orders {
		compositeRecords[i] = []interface{}{
			nil, // id will be generated
			o.CustomerID,
			o.DeliveryAddress,
			o.TotalPriceCents,
			o.TotalPriceCurrency.String(),
			pgtype.Timestamptz{Time: o.CreatedAt, Valid: true},
			pgtype.Timestamptz{Time: o.UpdatedAt, Valid: true},
		}
	}

	sql := `
		INSERT INTO orders (customer_id, delivery_address, total_price_cents, total_price_currency, created_at, updated_at)
		SELECT (unnest($1::v1_order[])).customer_id,
		       (unnest($1::v1_order[])).delivery_address,
		       (unnest($1::v1_order[])).total_price_cents,
		       (unnest($1::v1_order[])).total_price_currency,
		       (unnest($1::v1_order[])).created_at,
		       (unnest($1::v1_order[])).updated_at
		RETURNING id, customer_id, delivery_address, total_price_cents, total_price_currency, created_at, updated_at
	`

	rows, err := r.conn.Query(ctx, sql, compositeRecords)
	if err != nil {
		return nil, fmt.Errorf("failed to bulk insert orders: %w", err)
	}
	defer rows.Close()

	var result []order.Order
	i := 0
	for rows.Next() {
		var dal OrderDal
		var createdAt, updatedAt pgtype.Timestamptz

		err := rows.Scan(
			&dal.Id,
			&dal.CustomerId,
			&dal.DeliveryAddress,
			&dal.TotalPriceCents,
			&dal.TotalPriceCurrency,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan iorderrepo: %w", err)
		}

		dal.CreatedAt = createdAt.Time
		dal.UpdatedAt = updatedAt.Time

		model, err := dal.ToModel()
		if err != nil {
			return nil, fmt.Errorf("failed to convert iorderrepo dal to model: %w", err)
		}

		model.OrderItems = append(model.OrderItems, orders[i].OrderItems...)
		i++

		result = append(result, *model)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return result, nil
}

// Query retrieves orders based on filter criteria.
func (r *PostgresOrderRepository) Query(
	ctx context.Context,
	filter *order.QueryOrdersModel,
) ([]order.Order, error) {
	ctx, span := otel.Tracer("dal").Start(ctx, "DAL.GetOrders")
	defer span.End()

	query := r.sb.
		Select(
			"id",
			"customer_id",
			"delivery_address",
			"total_price_cents",
			"total_price_currency",
			"created_at",
			"updated_at",
		).
		From("orders")

	if len(filter.Ids) > 0 {
		query = query.Where(sq.Eq{"id": filter.Ids})
	}

	if len(filter.CustomerIds) > 0 {
		query = query.Where(sq.Eq{"customer_id": filter.CustomerIds})
	}

	if filter.Limit > 0 {
		query = query.Limit(uint64(filter.Limit))
	}

	if filter.Offset > 0 {
		query = query.Offset(uint64(filter.Offset))
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := r.conn.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %w", err)
	}
	defer rows.Close()

	var result []order.Order
	for rows.Next() {
		var dal OrderDal
		var createdAt, updatedAt pgtype.Timestamptz

		err := rows.Scan(
			&dal.Id,
			&dal.CustomerId,
			&dal.DeliveryAddress,
			&dal.TotalPriceCents,
			&dal.TotalPriceCurrency,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan iorderrepo: %w", err)
		}

		dal.CreatedAt = createdAt.Time
		dal.UpdatedAt = updatedAt.Time

		model, err := dal.ToModel()
		if err != nil {
			return nil, fmt.Errorf("failed to convert iorderrepo dal to model: %w", err)
		}
		result = append(result, *model)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return result, nil
}
