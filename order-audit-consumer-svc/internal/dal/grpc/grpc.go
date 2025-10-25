package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/corray333/backend-labs/consumer/internal/service/models"
	pb "github.com/corray333/backend-labs/order/pkg/api/v1"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Client represents a gRPC client for order service.
type Client struct {
	conn   *grpc.ClientConn
	client pb.OrderServiceClient
}

// MustNewClient creates a new gRPC client.
func MustNewClient() *Client {
	addr := viper.GetString("grpc.order_service_addr")
	if addr == "" {
		panic("grpc.order_service_addr is not set in config")
	}

	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to order service: %v", err))
	}

	client := pb.NewOrderServiceClient(conn)

	slog.Info("gRPC client connected to order service", "address", addr)

	return &Client{
		conn:   conn,
		client: client,
	}
}

// Close closes the gRPC connection.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}

	return nil
}

// SaveAuditLogs sends audit logs to the order service via gRPC.
func (c *Client) SaveAuditLogs(ctx context.Context, auditLogs []models.AuditLogOrder) error {
	ctx, span := otel.Tracer("grpc-client").Start(ctx, "Client.SaveAuditLogs")
	defer span.End()

	timeout := viper.GetInt("grpc.timeout_seconds")
	if timeout == 0 {
		timeout = 30
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	pbAuditLogs := make([]*pb.AuditLogOrder, len(auditLogs))
	for i, log := range auditLogs {
		pbAuditLogs[i] = &pb.AuditLogOrder{
			OrderId:     log.OrderID,
			OrderItemId: log.OrderItemID,
			CustomerId:  log.CustomerID,
			OrderStatus: log.OrderStatus,
			CreatedAt:   timestamppb.New(log.CreatedAt),
			UpdatedAt:   timestamppb.New(log.UpdatedAt),
		}
	}

	req := &pb.SaveAuditLogRequest{
		AuditLogs: pbAuditLogs,
	}

	resp, err := c.client.SaveAuditLog(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to save audit logs via gRPC: %w", err)
	}

	slog.Info("Audit logs saved via gRPC", "count", len(resp.AuditLogs))

	return nil
}
