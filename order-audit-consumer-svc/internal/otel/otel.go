package otel

import (
	"context"

	"github.com/corray333/backend-labs/consumer/internal/jaeger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

type OtelController struct {
	traceProvider *sdktrace.TracerProvider
}

func MustInitOtel() *OtelController {
	jaegerExporter := jaeger.MustNewJaeger()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(jaegerExporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("consumer-svc"),
		)),
	)

	otel.SetTracerProvider(tp)

	return &OtelController{
		traceProvider: tp,
	}
}

func (o *OtelController) Shutdown() error {
	if err := o.traceProvider.Shutdown(context.Background()); err != nil {
		return err
	}

	return nil
}
