// Package telemetry wires OpenTelemetry tracing and Prometheus metrics into
// the application. Both are optional and self-gate on configuration so the
// service runs unchanged in environments where a collector is not present.
package telemetry

import (
	"context"
	"fmt"

	"boilerplate-api/lib/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
)

// Tracer wraps the configured TracerProvider so downstream code can create
// spans (t.Start(ctx, name)) without importing the otel global directly.
type Tracer struct {
	trace.Tracer
	provider *sdktrace.TracerProvider
	Enabled  bool
}

// NewTracer configures OpenTelemetry.
//
// W3C trace-context and baggage propagators are always installed so trace
// context flows across HTTP/gRPC/event boundaries even when this service does
// not export its own spans. Span export is enabled only when
// OTEL_EXPORTER_OTLP_ENDPOINT is set (host:port of an OTLP/gRPC collector).
func NewTracer(lc fx.Lifecycle, env config.Env, logger config.Logger) (Tracer, error) {
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(env.ServiceName),
			semconv.ServiceVersion(env.ServiceVersion),
			attribute.String("deployment.environment", env.Environment),
		),
	)
	if err != nil {
		return Tracer{}, fmt.Errorf("telemetry: build resource: %w", err)
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(env.OtelTraceSampleRatio))),
	}

	enabled := env.OtelExporterEndpoint != ""
	if enabled {
		exp, err := otlptracegrpc.New(
			context.Background(),
			otlptracegrpc.WithEndpoint(env.OtelExporterEndpoint),
			otlptracegrpc.WithInsecure(),
		)
		if err != nil {
			return Tracer{}, fmt.Errorf("telemetry: create OTLP exporter: %w", err)
		}
		opts = append(opts, sdktrace.WithBatcher(exp))
	}

	tp := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)

	lc.Append(fx.Hook{
		// Flush buffered spans on shutdown so nothing is lost on redeploy.
		OnStop: func(ctx context.Context) error {
			return tp.Shutdown(ctx)
		},
	})

	if enabled {
		logger.Info("OpenTelemetry tracing exporting to ", env.OtelExporterEndpoint)
	} else {
		logger.Info("OpenTelemetry tracing: propagation-only (set OTEL_EXPORTER_OTLP_ENDPOINT to export spans)")
	}

	return Tracer{
		Tracer:   tp.Tracer(env.ServiceName),
		provider: tp,
		Enabled:  enabled,
	}, nil
}
