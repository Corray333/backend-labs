package order

import "time"

// Order represents an order message received from RabbitMQ.
type Order struct {
	ID              int64       `json:"id"`
	CustomerID      int64       `json:"customerId"`
	DeliveryAddress string      `json:"deliveryAddress"`
	OrderItems      []OrderItem `json:"orderItems"`
}

// OrderItem represents an item within an order.
type OrderItem struct {
	ID        int64     `json:"id"`
	OrderID   int64     `json:"orderId"`
	ProductID int64     `json:"productId"`
	Quantity  int       `json:"quantity"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
