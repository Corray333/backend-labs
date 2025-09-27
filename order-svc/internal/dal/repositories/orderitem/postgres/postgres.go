package postgresrepo

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/corray333/backend-labs/order/internal/service/models/currency"
	"github.com/corray333/backend-labs/order/internal/service/models/orderitem"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
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
	conn sqlx.ExtContext
}

// NewPostgresOrderItemRepository creates a new Postgres iorderrepo item repository.
func NewPostgresOrderItemRepository(pgClient sqlx.ExtContext) *PostgresOrderItemRepository {
	return &PostgresOrderItemRepository{
		conn: pgClient,
	}
}

// BulkInsert inserts multiple iorderrepo items and returns the inserted iorderrepo items with IDs.
func (r *PostgresOrderItemRepository) BulkInsert(
	ctx context.Context,
	orderItems []orderitem.OrderItem,
) ([]orderitem.OrderItem, error) {
	if len(orderItems) == 0 {
		return []orderitem.OrderItem{}, nil
	}

	sql := `
		INSERT INTO order_items (
			order_id,
			product_id,
			quantity,
			product_title,
			product_url,
			price_cents,
			price_currency,
			created_at,
			updated_at
		)
		SELECT
			order_id,
			product_id,
			quantity,
			product_title,
			product_url,
			price_cents,
			price_currency,
			created_at,
			updated_at
		FROM unnest($1::bigint[], $2::bigint[], $3::int[], $4::text[], $5::text[], $6::bigint[], $7::text[], $8::timestamp[], $9::timestamp[])
		AS t(order_id, product_id, quantity, product_title, product_url, price_cents, price_currency, created_at, updated_at)
		RETURNING
			id,
			order_id,
			product_id,
			quantity,
			product_title,
			product_url,
			price_cents,
			price_currency,
			created_at,
			updated_at
	`

	// Prepare arrays for bulk insert
	orderIds := make([]int64, len(orderItems))
	productIds := make([]int64, len(orderItems))
	quantities := make([]int, len(orderItems))
	productTitles := make([]string, len(orderItems))
	productUrls := make([]string, len(orderItems))
	priceCents := make([]int64, len(orderItems))
	priceCurrencies := make([]string, len(orderItems))
	createdAts := make([]time.Time, len(orderItems))
	updatedAts := make([]time.Time, len(orderItems))

	for i, oi := range orderItems {
		orderIds[i] = oi.OrderID
		productIds[i] = oi.ProductID
		quantities[i] = oi.Quantity
		productTitles[i] = oi.ProductTitle
		productUrls[i] = oi.ProductUrl
		priceCents[i] = oi.PriceCents
		priceCurrencies[i] = oi.PriceCurrency.String()
		createdAts[i] = oi.CreatedAt
		updatedAts[i] = oi.UpdatedAt
	}

	rows, err := r.conn.QueryContext(ctx, sql,
		pq.Array(orderIds),
		pq.Array(productIds),
		pq.Array(quantities),
		pq.Array(productTitles),
		pq.Array(productUrls),
		pq.Array(priceCents),
		pq.Array(priceCurrencies),
		pq.Array(createdAts),
		pq.Array(updatedAts))
	if err != nil {
		return nil, fmt.Errorf("failed to bulk insert iorderrepo items: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("failed to close rows", "error", err)
		}
	}()

	var result []orderitem.OrderItem
	for rows.Next() {
		var dal OrderItemDal
		err := rows.Scan(
			&dal.Id,
			&dal.OrderId,
			&dal.ProductId,
			&dal.Quantity,
			&dal.ProductTitle,
			&dal.ProductUrl,
			&dal.PriceCents,
			&dal.PriceCurrency,
			&dal.CreatedAt,
			&dal.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan iorderrepo item: %w", err)
		}
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
	sqlBuilder := strings.Builder{}
	sqlBuilder.WriteString(`
		SELECT
			id,
			order_id,
			product_id,
			quantity,
			product_title,
			product_url,
			price_cents,
			price_currency,
			created_at,
			updated_at
		FROM order_items
	`)

	args := []interface{}{}
	conditions := []string{}
	argIndex := 1

	if len(filter.Ids) > 0 {
		conditions = append(conditions, fmt.Sprintf("id = ANY($%d)", argIndex))
		args = append(args, pq.Array(filter.Ids))
		argIndex++
	}

	if len(filter.OrderIds) > 0 {
		conditions = append(conditions, fmt.Sprintf("order_id = ANY($%d)", argIndex))
		args = append(args, pq.Array(filter.OrderIds))
		argIndex++
	}

	if len(filter.ProductIds) > 0 {
		conditions = append(conditions, fmt.Sprintf("product_id = ANY($%d)", argIndex))
		args = append(args, pq.Array(filter.ProductIds))
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

	rows, err := r.conn.QueryContext(ctx, sqlBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query iorderrepo items: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("failed to close rows", "error", err)
		}
	}()

	var result []orderitem.OrderItem
	for rows.Next() {
		var dal OrderItemDal
		err := rows.Scan(
			&dal.Id,
			&dal.OrderId,
			&dal.ProductId,
			&dal.Quantity,
			&dal.ProductTitle,
			&dal.ProductUrl,
			&dal.PriceCents,
			&dal.PriceCurrency,
			&dal.CreatedAt,
			&dal.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan iorderrepo item: %w", err)
		}
		result = append(result, *dal.ToModel())
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return result, nil
}
