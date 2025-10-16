package tracing

import (
	"go.opentelemetry.io/otel/propagation"
)

type Config struct {
	ServiceName    string
	Environment    string
	JaegerEndpoint string
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

// func newTraceProvider(cfg Config, exporter sdktrace.SpanExporter) (*sdktrace.TracerProvider, error) {

// }
