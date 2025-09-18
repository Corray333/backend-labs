package createorder

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/corray333/backend-labs/order/internal/service/models/currency"
	"github.com/corray333/backend-labs/order/internal/service/models/order"
	"github.com/corray333/backend-labs/order/internal/service/models/orderitem"
	"github.com/go-playground/validator/v10"
)

// service is an interface for the service layer.
type service interface {
	BatchInsert(ctx context.Context, orders []order.Order) ([]order.Order, error)
}

// itemInCreateOrderRequest represents an item in a create order request.
type itemInCreateOrderRequest struct {
	ProductID     int64  `json:"productId"     validate:"gt=0"`
	Quantity      int    `json:"quantity"      validate:"gt=0"`
	ProductTitle  string `json:"productTitle"  validate:"required"`
	ProductUrl    string `json:"productUrl"`
	PriceCents    int64  `json:"priceCents"    validate:"gt=0"`
	PriceCurrency string `json:"priceCurrency" validate:"required"`
}

// toModel converts itemInCreateOrderRequest to orderitem.OrderItem.
func (r *itemInCreateOrderRequest) toModel() (*orderitem.OrderItem, error) {
	cur, err := currency.ParseCurrency(r.PriceCurrency)
	if err != nil {
		return nil, err
	}

	return &orderitem.OrderItem{
		ProductID:     r.ProductID,
		ProductUrl:    r.ProductUrl,
		PriceCents:    r.PriceCents,
		PriceCurrency: cur,
	}, nil
}

// orderInCreateOrderRequest represents an order in a create order request.
type orderInCreateOrderRequest struct {
	CustomerID         int64                      `json:"customerId"         validate:"gt=0"`
	DeliveryAddress    string                     `json:"deliveryAddress"    validate:"required"`
	TotalPriceCents    int64                      `json:"totalPriceCents"    validate:"gt=0"`
	TotalPriceCurrency string                     `json:"totalPriceCurrency" validate:"required"`
	CreatedAt          time.Time                  `json:"createdAt"`
	UpdatedAt          time.Time                  `json:"updatedAt"`
	OrderItems         []itemInCreateOrderRequest `json:"orderItems"         validate:"required,min=1,dive"`
}

// toModel converts orderInCreateOrderRequest to order.Order.
func (r *orderInCreateOrderRequest) toModel() (*order.Order, error) {
	cur, err := currency.ParseCurrency(r.TotalPriceCurrency)
	if err != nil {
		return nil, err
	}
	items := make([]orderitem.OrderItem, len(r.OrderItems))
	for i := range r.OrderItems {
		item, err := r.OrderItems[i].toModel()
		if err != nil {
			return nil, err
		}
		items[i] = *item
	}

	return &order.Order{
		CustomerID:         r.CustomerID,
		DeliveryAddress:    r.DeliveryAddress,
		TotalPriceCents:    r.TotalPriceCents,
		TotalPriceCurrency: cur,
		CreatedAt:          r.CreatedAt,
		UpdatedAt:          r.UpdatedAt,
		OrderItems:         items,
	}, nil
}

// createOrderRequest represents a create order request.
type createOrderRequest struct {
	Orders []orderInCreateOrderRequest `json:"orders" validate:"required,min=1,dive"`
}

// Validate validates the create order request.
func (r *createOrderRequest) Validate() error {
	return validator.New().Struct(r)
}

// BatchInsert handles the batch insert request.
func BatchInsert(w http.ResponseWriter, r *http.Request, service service) {
	ordersReq := createOrderRequest{}
	if err := json.NewDecoder(r.Body).Decode(&ordersReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		slog.Error("Error decoding request body for batch insert", "error", err)

		return
	}

	if err := ordersReq.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		slog.Error("Error validating request body for batch insert", "error", err)

		return
	}

	orders := make([]order.Order, len(ordersReq.Orders))
	for i, req := range ordersReq.Orders {
		model, err := req.toModel()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			slog.Error("Error converting model request to model", "error", err)

			return
		}
		orders[i] = *model
	}

	insertedOrders, err := service.BatchInsert(r.Context(), orders)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("Error performing batch insert", "error", err)

		return
	}

	if err := json.NewEncoder(w).Encode(insertedOrders); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("Error sending response for batch insert", "error", err)
	}
}
