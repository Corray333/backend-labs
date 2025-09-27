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
	"github.com/corray333/backend-labs/order/internal/service/services/ordersvc"
	httptransport "github.com/corray333/backend-labs/order/internal/transport/http"
)

// App represents the application.
type App struct {
	orderSvc       *ordersvc.OrderService
	transport      *httptransport.HTTPTransport
	postgresClient *postgres.Client
	rabbitMqClient *rabbitmq.Client
}

// MustNewApp creates a new application.
func MustNewApp() *App {
	rabbitMqClient := rabbitmq.MustNewClient()
	auditRabbitMQRepository := audit.NewAuditRabbitMQRepository(rabbitMqClient)

	postgresClient := postgres.MustNewClient()

	orderSvc := ordersvc.MustNewOrderService(
		ordersvc.WithPostgresClient(postgresClient),
		ordersvc.WithAuditor(auditRabbitMQRepository),
	)

	transport := httptransport.NewHTTPTransport(orderSvc)
	transport.RegisterRoutes()

	return &App{
		orderSvc:       orderSvc,
		transport:      transport,
		postgresClient: postgresClient,
		rabbitMqClient: rabbitMqClient,
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

	<-stop
	slog.Info("Shutdown signal received")

	a.gracefulShutdown()
}

// gracefulShutdown performs graceful shutdown of all application components.
// It shuts down HTTP server, RabbitMQ and PostgreSQL connections in parallel.
func (a *App) gracefulShutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	wg.Go(func() {
		if err := a.transport.Shutdown(ctx); err != nil {
			slog.Error("HTTP server shutdown error", "error", err)
		} else {
			slog.Info("HTTP server stopped gracefully")
		}
	})

	wg.Go(func() {
		if err := a.rabbitMqClient.Close(); err != nil {
			slog.Error("RabbitMQ connection close error", "error", err)
		} else {
			slog.Info("RabbitMQ connection closed gracefully")
		}
	})

	wg.Go(func() {
		if err := a.postgresClient.Close(); err != nil {
			slog.Error("Database connection close error", "error", err)
		} else {
			slog.Info("Database connection closed gracefully")
		}
	})

	wg.Wait()
	slog.Info("Application shutdown complete")
}
