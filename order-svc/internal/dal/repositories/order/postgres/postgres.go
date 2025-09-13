package postgresrepo

import (
	"context"
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"github.com/corray333/backend-labs/order/internal/service/models/currency"
	"github.com/corray333/backend-labs/order/internal/service/models/order"
	"github.com/corray333/backend-labs/order/internal/service/models/orderitem"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// OrderDal represents order data access layer model
type OrderDal struct {
	Id                 int64     `db:"id"`
	CustomerId         int64     `db:"customer_id"`
	DeliveryAddress    string    `db:"delivery_address"`
	TotalPriceCents    int64     `db:"total_price_cents"`
	TotalPriceCurrency string    `db:"total_price_currency"`
	CreatedAt          time.Time `db:"created_at"`
	UpdatedAt          time.Time `db:"updated_at"`
}

// ToModel converts OrderDal to service layer Order model
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

// FromModel converts service layer Order model to OrderDal
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

// OrderArray is a custom type for PostgreSQL array handling
type OrderArray []*OrderDal

// Value implements driver.Valuer interface for PostgreSQL array
func (a OrderArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return nil, nil
	}

	var values []string
	for _, order := range a {
		values = append(values, fmt.Sprintf("(%d,'%s',%d,'%s','%s','%s')",
			order.CustomerId,
			order.DeliveryAddress,
			order.TotalPriceCents,
			order.TotalPriceCurrency,
			order.CreatedAt.Format(time.RFC3339),
			order.UpdatedAt.Format(time.RFC3339)))
	}

	return strings.Join(values, ","), nil
}

type PostgresOrderRepository struct {
	conn sqlx.ExtContext
}

func NewPostgresOrderRepository(pgClient sqlx.ExtContext) *PostgresOrderRepository {
	return &PostgresOrderRepository{
		conn: pgClient,
	}
}

// BulkInsert inserts multiple orders and returns the inserted orders with IDs
func (r *PostgresOrderRepository) BulkInsert(ctx context.Context, orders []order.Order) ([]order.Order, error) {
	if len(orders) == 0 {
		return []order.Order{}, nil
	}

	sql := `
		INSERT INTO orders (
			customer_id,
			delivery_address,
			total_price_cents,
			total_price_currency,
			created_at,
			updated_at
		)
		SELECT
			customer_id,
			delivery_address,
			total_price_cents,
			total_price_currency,
			created_at,
			updated_at
		FROM unnest($1::bigint[], $2::text[], $3::bigint[], $4::text[], $5::timestamp[], $6::timestamp[])
		AS t(customer_id, delivery_address, total_price_cents, total_price_currency, created_at, updated_at)
		RETURNING
			id,
			customer_id,
			delivery_address,
			total_price_cents,
			total_price_currency,
			created_at,
			updated_at
	`

	customerIds := make([]int64, len(orders))
	deliveryAddresses := make([]string, len(orders))
	totalPriceCents := make([]int64, len(orders))
	totalPriceCurrencies := make([]string, len(orders))
	createdAts := make([]time.Time, len(orders))
	updatedAts := make([]time.Time, len(orders))

	for i, o := range orders {
		customerIds[i] = o.CustomerID
		deliveryAddresses[i] = o.DeliveryAddress
		totalPriceCents[i] = o.TotalPriceCents
		totalPriceCurrencies[i] = o.TotalPriceCurrency.String()
		createdAts[i] = o.CreatedAt
		updatedAts[i] = o.UpdatedAt
	}

	rows, err := r.conn.QueryContext(ctx, sql,
		pq.Array(customerIds),
		pq.Array(deliveryAddresses),
		pq.Array(totalPriceCents),
		pq.Array(totalPriceCurrencies),
		pq.Array(createdAts),
		pq.Array(updatedAts))
	if err != nil {
		return nil, fmt.Errorf("failed to bulk insert orders: %w", err)
	}
	defer rows.Close()

	var result []order.Order
	i := 0
	for rows.Next() {
		dal := OrderDal{}
		err := rows.Scan(
			&dal.Id,
			&dal.CustomerId,
			&dal.DeliveryAddress,
			&dal.TotalPriceCents,
			&dal.TotalPriceCurrency,
			&dal.CreatedAt,
			&dal.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		model, err := dal.ToModel()
		if err != nil {
			return nil, fmt.Errorf("failed to convert order dal to model: %w", err)
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

// Query retrieves orders based on filter criteria
func (r *PostgresOrderRepository) Query(ctx context.Context, filter *order.QueryOrdersModel) ([]order.Order, error) {
	sqlBuilder := strings.Builder{}
	sqlBuilder.WriteString(`
		SELECT
			id,
			customer_id,
			delivery_address,
			total_price_cents,
			total_price_currency,
			created_at,
			updated_at
		FROM orders
	`)

	args := []interface{}{}
	conditions := []string{}
	argIndex := 1

	if len(filter.Ids) > 0 {
		conditions = append(conditions, fmt.Sprintf("id = ANY($%d)", argIndex))
		args = append(args, pq.Array(filter.Ids))
		argIndex++
	}

	if len(filter.CustomerIds) > 0 {
		conditions = append(conditions, fmt.Sprintf("customer_id = ANY($%d)", argIndex))
		args = append(args, pq.Array(filter.CustomerIds))
		argIndex++
	}

	if len(conditions) > 0 {
		sqlBuilder.WriteString(" WHERE " + strings.Join(conditions, " AND "))
	}

	if filter.Limit > 0 {
		sqlBuilder.WriteString(fmt.Sprintf(" LIMIT $%d", argIndex))
		args = append(args, filter.Limit)
		argIndex++
	}

	if filter.Offset > 0 {
		sqlBuilder.WriteString(fmt.Sprintf(" OFFSET $%d", argIndex))
		args = append(args, filter.Offset)
	}

	fmt.Println(sqlBuilder.String())

	rows, err := r.conn.QueryContext(ctx, sqlBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %w", err)
	}
	defer rows.Close()

	var result []order.Order
	for rows.Next() {
		var dal OrderDal
		err := rows.Scan(
			&dal.Id,
			&dal.CustomerId,
			&dal.DeliveryAddress,
			&dal.TotalPriceCents,
			&dal.TotalPriceCurrency,
			&dal.CreatedAt,
			&dal.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		model, err := dal.ToModel()
		if err != nil {
			return nil, fmt.Errorf("failed to convert order dal to model: %w", err)
		}
		result = append(result, *model)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return result, nil
}
