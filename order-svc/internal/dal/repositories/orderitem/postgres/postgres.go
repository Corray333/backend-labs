package postgresrepo

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/corray333/backend-labs/order/internal/service/models/currency"
	"github.com/corray333/backend-labs/order/internal/service/models/orderitem"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// OrderItemDal represents iorderrepo item data access layer model.
type OrderItemDal struct {
	Id            int64     `db:"id"`
	OrderId       int64     `db:"order_id"`
	ProductId     int64     `db:"product_id"`
	Quantity      int       `db:"quantity"`
	ProductTitle  string    `db:"product_title"`
	ProductUrl    string    `db:"product_url"`
	PriceCents    int64     `db:"price_cents"`
	PriceCurrency string    `db:"price_currency"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

// ToModel converts OrderItemDal to service layer OrderItem model.
func (oi *OrderItemDal) ToModel() *orderitem.OrderItem {
	cur, err := currency.ParseCurrency(oi.PriceCurrency)
	if err != nil {
		return nil
	}

	return &orderitem.OrderItem{
		ID:            oi.Id,
		OrderID:       oi.OrderId,
		ProductID:     oi.ProductId,
		Quantity:      oi.Quantity,
		ProductTitle:  oi.ProductTitle,
		ProductUrl:    oi.ProductUrl,
		PriceCents:    oi.PriceCents,
		PriceCurrency: cur,
		CreatedAt:     oi.CreatedAt,
		UpdatedAt:     oi.UpdatedAt,
	}
}

// OrderItemDalFromModel converts service layer OrderItem model to OrderItemDal.
func OrderItemDalFromModel(oi *orderitem.OrderItem) *OrderItemDal {
	return &OrderItemDal{
		Id:            oi.ID,
		OrderId:       oi.OrderID,
		ProductId:     oi.ProductID,
		Quantity:      oi.Quantity,
		ProductTitle:  oi.ProductTitle,
		ProductUrl:    oi.ProductUrl,
		PriceCents:    oi.PriceCents,
		PriceCurrency: oi.PriceCurrency.String(),
		CreatedAt:     oi.CreatedAt,
		UpdatedAt:     oi.UpdatedAt,
	}
}

// PostgresOrderItemRepository represents a Postgres iorderrepo item repository.
type PostgresOrderItemRepository struct {
	conn GenericConn
	sb   sq.StatementBuilderType
}

// GenericConn is an interface that works with both pgxpool.Pool and pgx.Tx
type GenericConn interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

// NewPostgresOrderItemRepository creates a new Postgres iorderrepo item repository.
func NewPostgresOrderItemRepository(conn GenericConn) *PostgresOrderItemRepository {
	return &PostgresOrderItemRepository{
		conn: conn,
		sb:   sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

// BulkInsert inserts multiple iorderrepo items and returns the inserted iorderrepo items with IDs.
// Uses PostgreSQL composite type v1_order_item for optimization.
func (r *PostgresOrderItemRepository) BulkInsert(
	ctx context.Context,
	orderItems []orderitem.OrderItem,
) ([]orderitem.OrderItem, error) {
	if len(orderItems) == 0 {
		return []orderitem.OrderItem{}, nil
	}

	// Prepare array of composite type values
	compositeRecords := make([][]interface{}, len(orderItems))
	for i, oi := range orderItems {
		compositeRecords[i] = []interface{}{
			nil, // id will be generated
			oi.OrderID,
			oi.ProductID,
			oi.Quantity,
			oi.ProductTitle,
			oi.ProductUrl,
			oi.PriceCents,
			oi.PriceCurrency.String(),
			pgtype.Timestamptz{Time: oi.CreatedAt, Valid: true},
			pgtype.Timestamptz{Time: oi.UpdatedAt, Valid: true},
		}
	}

	// Use unnest with composite type for efficient bulk insert
	sql := `
		INSERT INTO order_items (order_id, product_id, quantity, product_title, product_url, price_cents, price_currency, created_at, updated_at)
		SELECT (unnest($1::v1_order_item[])).order_id,
		       (unnest($1::v1_order_item[])).product_id,
		       (unnest($1::v1_order_item[])).quantity,
		       (unnest($1::v1_order_item[])).product_title,
		       (unnest($1::v1_order_item[])).product_url,
		       (unnest($1::v1_order_item[])).price_cents,
		       (unnest($1::v1_order_item[])).price_currency,
		       (unnest($1::v1_order_item[])).created_at,
		       (unnest($1::v1_order_item[])).updated_at
		RETURNING id, order_id, product_id, quantity, product_title, product_url, price_cents, price_currency, created_at, updated_at
	`

	rows, err := r.conn.Query(ctx, sql, compositeRecords)
	if err != nil {
		return nil, fmt.Errorf("failed to bulk insert iorderrepo items: %w", err)
	}
	defer rows.Close()

	var result []orderitem.OrderItem
	for rows.Next() {
		var dal OrderItemDal
		var createdAt, updatedAt pgtype.Timestamptz

		err := rows.Scan(
			&dal.Id,
			&dal.OrderId,
			&dal.ProductId,
			&dal.Quantity,
			&dal.ProductTitle,
			&dal.ProductUrl,
			&dal.PriceCents,
			&dal.PriceCurrency,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan iorderrepo item: %w", err)
		}

		dal.CreatedAt = createdAt.Time
		dal.UpdatedAt = updatedAt.Time

		result = append(result, *dal.ToModel())
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return result, nil
}

// Query retrieves iorderrepo items based on filter criteria.
func (r *PostgresOrderItemRepository) Query(
	ctx context.Context,
	filter *orderitem.QueryOrderItemsModel,
) ([]orderitem.OrderItem, error) {
	query := r.sb.
		Select(
			"id",
			"order_id",
			"product_id",
			"quantity",
			"product_title",
			"product_url",
			"price_cents",
			"price_currency",
			"created_at",
			"updated_at",
		).
		From("order_items")

	if len(filter.Ids) > 0 {
		query = query.Where(sq.Eq{"id": filter.Ids})
	}

	if len(filter.OrderIds) > 0 {
		query = query.Where(sq.Eq{"order_id": filter.OrderIds})
	}

	if len(filter.ProductIds) > 0 {
		query = query.Where(sq.Eq{"product_id": filter.ProductIds})
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
		return nil, fmt.Errorf("failed to query iorderrepo items: %w", err)
	}
	defer rows.Close()

	var result []orderitem.OrderItem
	for rows.Next() {
		var dal OrderItemDal
		var createdAt, updatedAt pgtype.Timestamptz

		err := rows.Scan(
			&dal.Id,
			&dal.OrderId,
			&dal.ProductId,
			&dal.Quantity,
			&dal.ProductTitle,
			&dal.ProductUrl,
			&dal.PriceCents,
			&dal.PriceCurrency,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan iorderrepo item: %w", err)
		}

		dal.CreatedAt = createdAt.Time
		dal.UpdatedAt = updatedAt.Time

		result = append(result, *dal.ToModel())
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return result, nil
}
