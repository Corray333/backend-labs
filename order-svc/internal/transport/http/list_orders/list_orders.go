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

type service interface {
	GetOrders(ctx context.Context, model orderitem.QueryOrderItemsModel) ([]order.Order, error)
}

type queryOrdersRequest struct {
	Ids         []int64 `schema:"ids,omitempty"`
	CustomerIds []int64 `schema:"customerIds,omitempty"`
	Limit       int     `schema:"limit,omitempty"`
	Offset      int     `schema:"offset,omitempty"`
}

func (q *queryOrdersRequest) ToModel() orderitem.QueryOrderItemsModel {
	return orderitem.QueryOrderItemsModel{
		Ids:         q.Ids,
		CustomerIds: q.CustomerIds,
		Limit:       q.Limit,
		Offset:      q.Offset,
	}
}

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
