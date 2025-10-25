package jaeger

import (
	"go.opentelemetry.io/otel/exporters/jaeger"
)

func MustNewJaeger() *jaeger.Exporter {
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(
		jaeger.WithEndpoint("http://jaeger:14268/api/traces"),
	))
	if err != nil {
		panic(err)
	}

	return exp
}
