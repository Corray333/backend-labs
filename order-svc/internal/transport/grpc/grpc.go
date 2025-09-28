package grpctransport

import (
	"context"
	"log/slog"
	"net"
	"time"

	"github.com/corray333/backend-labs/order/internal/service/models/auditlog"
	"github.com/corray333/backend-labs/order/internal/service/models/order"
	"github.com/corray333/backend-labs/order/internal/service/models/orderitem"
	pb "github.com/corray333/backend-labs/order/pkg/api/v1"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// service is an interface for the service layer.
type service interface {
	GetOrders(ctx context.Context, model orderitem.QueryOrderItemsModel) ([]order.Order, error)
	BatchInsert(ctx context.Context, orders []order.Order) ([]order.Order, error)
	SaveAuditLogs(
		ctx context.Context,
		auditLogs []auditlog.AuditLogOrder,
	) ([]auditlog.AuditLogOrder, error)
}

// GRPCTransport represents the gRPC transport layer.
type GRPCTransport struct {
	server      *grpc.Server
	listener    net.Listener
	service     service
	orderServer *OrderServer
}

// NewGRPCTransport creates a new GRPCTransport.
func NewGRPCTransport(service service) *GRPCTransport {
	listener, err := net.Listen("tcp", ":"+viper.GetString("server.grpc.port"))
	if err != nil {
		panic(err)
	}

	server := newGRPCServer()
	orderServer := NewOrderServer(service)

	return &GRPCTransport{
		server:      server,
		listener:    listener,
		service:     service,
		orderServer: orderServer,
	}
}

// Run starts the gRPC server.
func (g *GRPCTransport) Run() error {
	g.RegisterServices()
	slog.Info("Starting gRPC server", "address", g.listener.Addr().String())

	return g.server.Serve(g.listener)
}

// Shutdown gracefully shuts down the gRPC server.
func (g *GRPCTransport) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		g.server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		g.server.Stop()

		return ctx.Err()
	}
}

// RegisterServices registers the gRPC services.
func (g *GRPCTransport) RegisterServices() {
	pb.RegisterOrderServiceServer(g.server, g.orderServer)
}

// newGRPCServer creates a new gRPC server with default settings.
func newGRPCServer() *grpc.Server {
	keepaliveParams := keepalive.ServerParameters{
		MaxConnectionIdle: time.Duration(
			viper.GetInt("server.grpc.keepalive.max_connection_idle"),
		) * time.Minute,
		MaxConnectionAge: time.Duration(
			viper.GetInt("server.grpc.keepalive.max_connection_age"),
		) * time.Minute,
		MaxConnectionAgeGrace: time.Duration(
			viper.GetInt("server.grpc.keepalive.max_connection_age_grace"),
		) * time.Second,
		Time: time.Duration(
			viper.GetInt("server.grpc.keepalive.time"),
		) * time.Second,
		Timeout: time.Duration(
			viper.GetInt("server.grpc.keepalive.timeout"),
		) * time.Second,
	}

	keepalivePolicy := keepalive.EnforcementPolicy{
		MinTime: time.Duration(
			viper.GetInt("server.grpc.keepalive.min_time"),
		) * time.Second,
		PermitWithoutStream: viper.GetBool("server.grpc.keepalive.permit_without_stream"),
	}

	opts := []grpc.ServerOption{
		grpc.KeepaliveParams(keepaliveParams),
		grpc.KeepaliveEnforcementPolicy(keepalivePolicy),
	}

	return grpc.NewServer(opts...)
}
