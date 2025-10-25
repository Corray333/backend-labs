package app

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/corray333/backend-labs/order/internal/dal/postgres"
	"github.com/corray333/backend-labs/order/internal/dal/rabbitmq"
	"github.com/corray333/backend-labs/order/internal/dal/repositories/audit"
	"github.com/corray333/backend-labs/order/internal/otel"
	"github.com/corray333/backend-labs/order/internal/service/services/ordersvc"
	grpctransport "github.com/corray333/backend-labs/order/internal/transport/grpc"
	httptransport "github.com/corray333/backend-labs/order/internal/transport/http"
)

// App represents the application.
type App struct {
	orderSvc       *ordersvc.OrderService
	transport      *httptransport.HTTPTransport
	postgresClient *postgres.Client
	rabbitMqClient *rabbitmq.Client
	grpcTransport  *grpctransport.GRPCTransport
	otelController *otel.OtelController
}

// MustNewApp creates a new application.
func MustNewApp() *App {
	otelController := otel.MustInitOtel()
	rabbitMqClient := rabbitmq.MustNewClient()
	postgresClient := postgres.MustNewClient()
	auditRabbitMQRepository := audit.NewAuditRabbitMQRepository(rabbitMqClient, postgresClient)

	orderSvc := ordersvc.MustNewOrderService(
		ordersvc.WithPostgresClient(postgresClient),
		ordersvc.WithAuditor(auditRabbitMQRepository),
	)

	transport := httptransport.NewHTTPTransport(orderSvc)
	transport.RegisterRoutes()

	grpcTransport := grpctransport.NewGRPCTransport(orderSvc)

	return &App{
		orderSvc:       orderSvc,
		transport:      transport,
		postgresClient: postgresClient,
		rabbitMqClient: rabbitMqClient,
		grpcTransport:  grpcTransport,
		otelController: otelController,
	}
}

// Run starts the application.
// Tracks interrupt signal to gracefully shut down the application.
func (a *App) Run() {
	// Create a channel to receive OS signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("Starting HTTP server")
		if err := a.transport.Run(); err != nil {
			slog.Error("HTTP server error", "error", err)
		}
	}()

	go func() {
		slog.Info("Starting gRPC server")
		if err := a.grpcTransport.Run(); err != nil {
			slog.Error("gRPC server error", "error", err)
		}
	}()

	<-stop
	slog.Info("Shutdown signal received")

	a.gracefulShutdown()
}

// gracefulShutdown performs graceful shutdown of all application components.
// It shuts down HTTP server, RabbitMQ and PostgreSQL connections in parallel.
func (a *App) gracefulShutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.transport.Shutdown(ctx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	} else {
		slog.Info("HTTP server stopped gracefully")
	}

	var wg sync.WaitGroup

	wg.Go(func() {
		if err := a.rabbitMqClient.Close(); err != nil {
			slog.Error("RabbitMQ connection close error", "error", err)
		} else {
			slog.Info("RabbitMQ connection closed gracefully")
		}
	})
	wg.Go(func() {
		if err := a.otelController.Shutdown(); err != nil {
			slog.Error("Otel trace provider connection close error", "error", err)
		} else {
			slog.Info("Otel trace provider connection closed gracefully")
		}
	})

	wg.Go(func() {
		if err := a.postgresClient.Close(); err != nil {
			slog.Error("Database connection close error", "error", err)
		} else {
			slog.Info("Database connection closed gracefully")
		}
	})

	wg.Go(func() {
		if err := a.grpcTransport.Shutdown(ctx); err != nil {
			slog.Error("gRPC server shutdown error", "error", err)
		} else {
			slog.Info("gRPC server stopped gracefully")
		}
	})

	wg.Wait()
	slog.Info("Application shutdown complete")
}
