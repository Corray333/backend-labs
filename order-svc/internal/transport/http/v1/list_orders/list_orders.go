package listorders

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/corray333/backend-labs/order/internal/service/models/order"
	"github.com/corray333/backend-labs/order/internal/service/models/orderitem"
	"github.com/corray333/backend-labs/order/internal/transport/http/v1/converters"
	pb "github.com/corray333/backend-labs/order/pkg/api/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

// service is an interface for the service layer.
type service interface {
	GetOrders(ctx context.Context, model orderitem.QueryOrderItemsModel) ([]order.Order, error)
}

// parseIntSlice parses comma-separated string to slice of int64.
func parseIntSlice(s string) []int64 {
	if s == "" {
		return nil
	}

	parts := make([]string, 0)
	current := ""
	for _, char := range s {
		if char == ',' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	result := make([]int64, 0, len(parts))
	for _, part := range parts {
		if val, err := strconv.ParseInt(part, 10, 64); err == nil {
			result = append(result, val)
		}
	}

	return result
}

// ListOrders handles the get orders request using protobuf types.
func ListOrders(w http.ResponseWriter, r *http.Request, service service) {
	// Parse query parameters into protobuf message
	query := r.URL.Query()

	listReq := &pb.ListOrdersRequest{
		Ids:         parseIntSlice(query.Get("ids")),
		CustomerIds: parseIntSlice(query.Get("customerIds")),
	}

	if limitStr := query.Get("limit"); limitStr != "" {
		if limit, err := strconv.ParseInt(limitStr, 10, 32); err == nil {
			listReq.Limit = int32(limit)
		}
	}

	if offsetStr := query.Get("offset"); offsetStr != "" {
		if offset, err := strconv.ParseInt(offsetStr, 10, 32); err == nil {
			listReq.Offset = int32(offset)
		}
	}

	// Convert protobuf to internal model
	queryModel := converters.ListOrdersRequestFromProto(listReq)

	// Call service
	orders, err := service.GetOrders(r.Context(), queryModel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error("Error getting orders", "error", err)

		return
	}

	// Convert response to protobuf and send as JSON
	response := converters.ListOrdersResponseToProto(orders)
	w.Header().Set("Content-Type", "application/json")

	jsonData, err := protojson.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		slog.Error("Error marshaling response for list orders", "error", err)

		return
	}

	if _, err := w.Write(jsonData); err != nil {
		slog.Error("Error writing response for list orders", "error", err)
	}
}
