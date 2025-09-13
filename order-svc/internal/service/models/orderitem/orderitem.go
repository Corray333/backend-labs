package orderitem

import (
	"time"

	"github.com/corray333/backend-labs/order/internal/service/models/currency"
)

// OrderItem represents an item within an order
type OrderItem struct {
	ID            int64             `json:"id"`
	OrderID       int64             `json:"orderId"`
	ProductID     int64             `json:"productId"`
	Quantity      int               `json:"quantity"`
	ProductTitle  string            `json:"productTitle"`
	ProductUrl    string            `json:"productUrl"`
	PriceCents    int64             `json:"priceCents"`
	PriceCurrency currency.Currency `json:"priceCurrency"`
	CreatedAt     time.Time         `json:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`
}
