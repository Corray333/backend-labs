package auditlog

import "time"

// AuditLogOrder represents an audit log entry for order operations.
type AuditLogOrder struct {
	ID          int64     `json:"id"`
	OrderID     int64     `json:"order_id"`
	OrderItemID int64     `json:"order_item_id"`
	CustomerID  int64     `json:"customer_id"`
	OrderStatus string    `json:"order_status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}