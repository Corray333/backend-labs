package order

import (
	"time"

	"github.com/corray333/backend-labs/order/internal/service/models/currency"
	"github.com/corray333/backend-labs/order/internal/service/models/orderitem"
)

// Order represents an iorderrepo in the system.
type Order struct {
	ID                 int64                 `json:"id"`
	CustomerID         int64                 `json:"customerId"`
	DeliveryAddress    string                `json:"deliveryAddress"`
	TotalPriceCents    int64                 `json:"totalPriceCents"`
	TotalPriceCurrency currency.Currency     `json:"totalPriceCurrency"`
	CreatedAt          time.Time             `json:"createdAt"`
	UpdatedAt          time.Time             `json:"updatedAt"`
	OrderItems         []orderitem.OrderItem `json:"orderItems"`
}
