package listorders

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/corray333/backend-labs/order/internal/service/models/order"
	"github.com/corray333/backend-labs/order/internal/service/models/orderitem"
	"github.com/gorilla/schema"
)

// service is an interface for the service layer.
type service interface {
	GetOrders(ctx context.Context, model orderitem.QueryOrderItemsModel) ([]order.Order, error)
}

// queryOrdersRequest represents the request for getting orders.
type queryOrdersRequest struct {
	Ids         []int64 `schema:"ids,omitempty"`
	CustomerIds []int64 `schema:"customerIds,omitempty"`
	Limit       int     `schema:"limit,omitempty"`
	Offset      int     `schema:"offset,omitempty"`
}

// ToModel converts queryOrdersRequest to orderitem.QueryOrderItemsModel.
func (q *queryOrdersRequest) ToModel() orderitem.QueryOrderItemsModel {
	return orderitem.QueryOrderItemsModel{
		Ids:         q.Ids,
		CustomerIds: q.CustomerIds,
		Limit:       q.Limit,
		Offset:      q.Offset,
	}
}

// ListOrders handles the get orders request.
func ListOrders(w http.ResponseWriter, r *http.Request, service service) {
	decoder := schema.NewDecoder()
	query := &queryOrdersRequest{}
	err := decoder.Decode(query, r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		slog.Error("Error decoding request", "error", err)

		return
	}

	orders, err := service.GetOrders(r.Context(), query.ToModel())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("Error getting orders", "error", err)

		return
	}

	if err := json.NewEncoder(w).Encode(orders); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("Error sending response", "error", err)
	}
}
