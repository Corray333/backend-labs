package app

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/corray333/backend-labs/consumer/internal/dal/postgres"
	auditrepo "github.com/corray333/backend-labs/consumer/internal/dal/repositories/audit/postgres"
	inboxrepo "github.com/corray333/backend-labs/consumer/internal/dal/repositories/inbox/postgres"
	"github.com/corray333/backend-labs/consumer/internal/otel"
	"github.com/corray333/backend-labs/consumer/internal/rabbitmq"
	"github.com/corray333/backend-labs/consumer/internal/service/services/consumersvc"
	"github.com/corray333/backend-labs/consumer/internal/transport/consumer"
	inboxworker "github.com/corray333/backend-labs/consumer/internal/worker/inbox"
)

// App represents the application.
type App struct {
	consumerSvc    *consumersvc.ConsumerService
	consumerTransp *consumer.Consumer
	inboxWorker    *inboxworker.Worker
	rabbitMqClient *rabbitmq.Client
	postgresClient *postgres.Client
	otelController *otel.OtelController
}

// MustNewApp creates a new application.
func MustNewApp() *App {
	otelController := otel.MustInitOtel()
	rabbitMqClient := rabbitmq.MustNewClient()
	postgresClient := postgres.MustNewClient()

	// Initialize PostgreSQL audit repository
	auditRepository := auditrepo.NewAuditRepository(postgresClient)

	// Initialize inbox repository
	inboxRepository := inboxrepo.NewInboxRepository(postgresClient)

	consumerSvc := consumersvc.MustNewConsumerService(
		consumersvc.WithAuditRepository(auditRepository),
	)

	consumerTransp := consumer.NewConsumer(rabbitMqClient, consumerSvc, inboxRepository)

	// Initialize inbox worker
	inboxWorker := inboxworker.NewWorker(inboxRepository, consumerSvc)

	return &App{
		consumerSvc:    consumerSvc,
		consumerTransp: consumerTransp,
		inboxWorker:    inboxWorker,
		rabbitMqClient: rabbitMqClient,
		postgresClient: postgresClient,
		otelController: otelController,
	}
}

// Run starts the application.
// Tracks interrupt signal to gracefully shut down the application.
func (a *App) Run() {
	// Create a channel to receive OS signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		slog.Info("Starting consumer")
		if err := a.consumerTransp.Run(ctx); err != nil {
			slog.Error("Consumer error", "error", err)
		}
	}()

	go func() {
		slog.Info("Starting inbox worker")
		a.inboxWorker.Start(ctx)
	}()

	<-stop
	slog.Info("Shutdown signal received")
	cancel()

	a.gracefulShutdown()
}

// gracefulShutdown performs graceful shutdown of all application components.
// It shuts down components sequentially: inbox worker, consumer, RabbitMQ, PostgreSQL, and OpenTelemetry.
func (a *App) gracefulShutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Stop inbox worker
	a.inboxWorker.Stop()
	slog.Info("Inbox worker stopped gracefully")

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

	a.postgresClient.Close()

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
