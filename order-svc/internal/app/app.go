package app

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/corray333/backend-labs/order/internal/dal/postgres"
	"github.com/corray333/backend-labs/order/internal/service/services/ordersvc"
	httptransport "github.com/corray333/backend-labs/order/internal/transport/http"
)

// App represents the application.
type App struct {
	orderSvc       *ordersvc.OrderService
	transport      *httptransport.HTTPTransport
	postgresClient *postgres.Client
}

// MustNewApp creates a new application.
func MustNewApp() *App {
	postgresClient := postgres.MustNewClient()

	orderSvc := ordersvc.MustNewOrderService(
		ordersvc.WithPostgresClient(postgresClient),
	)

	transport := httptransport.NewHTTPTransport(orderSvc)
	transport.RegisterRoutes()

	return &App{
		orderSvc:       orderSvc,
		transport:      transport,
		postgresClient: postgresClient,
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.transport.Shutdown(ctx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	} else {
		slog.Info("HTTP server stopped gracefully")
	}

	if err := a.postgresClient.Close(); err != nil {
		slog.Error("Database connection close error", "error", err)
	} else {
		slog.Info("Database connection closed gracefully")
	}

	slog.Info("Application shutdown complete")
}
