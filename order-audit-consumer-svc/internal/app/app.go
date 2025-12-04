package app

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpcclient "github.com/corray333/backend-labs/consumer/internal/dal/grpc"
	"github.com/corray333/backend-labs/consumer/internal/otel"
	"github.com/corray333/backend-labs/consumer/internal/rabbitmq"
	"github.com/corray333/backend-labs/consumer/internal/service/services/consumersvc"
	"github.com/corray333/backend-labs/consumer/internal/transport/consumer"
)

// App represents the application.
type App struct {
	consumerSvc    *consumersvc.ConsumerService
	consumerTransp *consumer.Consumer
	rabbitMqClient *rabbitmq.Client
	grpcClient     *grpcclient.Client
	otelController *otel.OtelController
}

// MustNewApp creates a new application.
func MustNewApp() *App {
	otelController := otel.MustInitOtel()
	rabbitMqClient := rabbitmq.MustNewClient()
	grpcClient := grpcclient.MustNewClient()

	consumerSvc := consumersvc.MustNewConsumerService(
		consumersvc.WithAuditRepository(grpcClient),
	)

	consumerTransp := consumer.NewConsumer(rabbitMqClient, consumerSvc)

	return &App{
		consumerSvc:    consumerSvc,
		consumerTransp: consumerTransp,
		rabbitMqClient: rabbitMqClient,
		grpcClient:     grpcClient,
		otelController: otelController,
	}
}

// Run starts the application.
// Tracks interrupt signal to gracefully shut down the application.
func (a *App) Run() {
	// Create a channel to receive OS signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	ctx := context.Background()

	go func() {
		slog.Info("Starting consumer")
		if err := a.consumerTransp.Run(ctx); err != nil {
			slog.Error("Consumer error", "error", err)
		}
	}()

	<-stop
	slog.Info("Shutdown signal received")

	a.gracefulShutdown()
}

// gracefulShutdown performs graceful shutdown of all application components.
// It shuts down components sequentially: consumer, RabbitMQ, gRPC client, and OpenTelemetry.
func (a *App) gracefulShutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.consumerTransp.Shutdown(); err != nil {
		slog.Error("Consumer shutdown error", "error", err)
	} else {
		slog.Info("Consumer stopped gracefully")
	}

	if err := a.rabbitMqClient.Close(); err != nil {
		slog.Error("RabbitMQ connection close error", "error", err)
	} else {
		slog.Info("RabbitMQ connection closed gracefully")
	}

	if err := a.grpcClient.Close(); err != nil {
		slog.Error("gRPC client close error", "error", err)
	} else {
		slog.Info("gRPC client closed gracefully")
	}

	if err := a.otelController.Shutdown(); err != nil {
		slog.Error("Otel trace provider connection close error", "error", err)
	} else {
		slog.Info("Otel trace provider connection closed gracefully")
	}

	select {
	case <-ctx.Done():
		slog.Warn("Shutdown timeout exceeded")
	default:
		slog.Info("Application shutdown complete")
	}
}
