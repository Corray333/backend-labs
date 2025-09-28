package createorder

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/corray333/backend-labs/order/internal/service/models/order"
	"github.com/corray333/backend-labs/order/internal/transport/http/v1/converters"
	pb "github.com/corray333/backend-labs/order/pkg/api/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

// service is an interface for the service layer.
type service interface {
	BatchInsert(ctx context.Context, orders []order.Order) ([]order.Order, error)
}

// BatchInsert handles the batch insert request using protobuf types.
func BatchInsert(w http.ResponseWriter, r *http.Request, service service) {
	var batchReq pb.BatchInsertRequest

	// Read and decode JSON request to protobuf message
	decoder := json.NewDecoder(r.Body)
	var jsonReq map[string]interface{}
	if err := decoder.Decode(&jsonReq); err != nil {
		http.Error(w, "Failed to decode request body", http.StatusBadRequest)
		slog.Error("Error decoding request body for batch insert", "error", err)

		return
	}

	jsonBytes, _ := json.Marshal(jsonReq)
	if err := protojson.Unmarshal(jsonBytes, &batchReq); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		slog.Error("Error unmarshaling request body for batch insert", "error", err)

		return
	}

	// Convert protobuf to internal models
	orders, err := converters.BatchInsertRequestFromProto(&batchReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		slog.Error("Error converting protobuf request to models", "error", err)

		return
	}

	// Call service
	insertedOrders, err := service.BatchInsert(r.Context(), orders)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("Error performing batch insert", "error", err)

		return
	}

	// Convert response to protobuf and send as JSON
	response := converters.BatchInsertResponseToProto(insertedOrders)
	w.Header().Set("Content-Type", "application/json")

	jsonData, err := protojson.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		slog.Error("Error marshaling response for batch insert", "error", err)

		return
	}

	if _, err := w.Write(jsonData); err != nil {
		slog.Error("Error writing response for batch insert", "error", err)
	}
}
