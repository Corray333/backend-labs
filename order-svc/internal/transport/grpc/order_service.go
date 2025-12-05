package grpctransport

import (
	"context"
	"log/slog"

	"github.com/corray333/backend-labs/order/internal/transport/http/v1/converters"
	pb "github.com/corray333/backend-labs/order/pkg/api/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// OrderServer implements the gRPC OrderService.
type OrderServer struct {
	pb.UnimplementedOrderServiceServer

	service service
}

// NewOrderServer creates a new OrderServer.
func NewOrderServer(service service) *OrderServer {
	return &OrderServer{
		service: service,
	}
}

// BatchInsert handles the batch insert gRPC request.
func (s *OrderServer) BatchInsert(
	ctx context.Context,
	req *pb.BatchInsertRequest,
) (*pb.BatchInsertResponse, error) {
	slog.Info("Received BatchInsert gRPC request", "orders_count", len(req.Orders))

	// Convert protobuf request to internal models
	orders, err := converters.BatchInsertRequestFromProto(req)
	if err != nil {
		slog.Error("Error converting protobuf request to models", "error", err)

		return nil, status.Errorf(codes.InvalidArgument, "failed to convert request: %v", err)
	}

	// Call service layer
	insertedOrders, err := s.service.BatchInsert(ctx, orders)
	if err != nil {
		slog.Error("Error performing batch insert", "error", err)

		return nil, status.Errorf(codes.Internal, "failed to insert orders: %v", err)
	}

	// Convert response to protobuf
	response := converters.BatchInsertResponseToProto(insertedOrders)

	slog.Info("BatchInsert completed successfully", "inserted_count", len(insertedOrders))

	return response, nil
}

// ListOrders handles the list orders gRPC request.
func (s *OrderServer) ListOrders(
	ctx context.Context,
	req *pb.ListOrdersRequest,
) (*pb.ListOrdersResponse, error) {
	slog.Info("Received ListOrders gRPC request",
		"ids", req.Ids,
		"customer_ids", req.CustomerIds,
		"limit", req.Limit,
		"offset", req.Offset)

	// Convert protobuf request to internal model
	queryModel := converters.ListOrdersRequestFromProto(req)

	// Call service layer
	orders, err := s.service.GetOrders(ctx, queryModel)
	if err != nil {
		slog.Error("Error getting orders", "error", err)

		return nil, status.Errorf(codes.Internal, "failed to get orders: %v", err)
	}

	// Convert response to protobuf
	response := converters.ListOrdersResponseToProto(orders)

	slog.Info("ListOrders completed successfully", "orders_count", len(orders))

	return response, nil
}

