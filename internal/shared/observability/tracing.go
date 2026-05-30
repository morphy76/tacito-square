package observability

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// InitTracer sets up OpenTelemetry tracing and metrics with both OTLP gRPC and Prometheus exporters.
// Returns a shutdown function that must be called on application exit.
func InitTracer(ctx context.Context, serviceName, serviceVersion, otelEndpoint string) (func(context.Context) error, error) {
	// 1. Initialize Prometheus pull-based exporter
	promExporter, err := otelprometheus.New()
	if err != nil {
		return nil, fmt.Errorf("creating Prometheus exporter: %w", err)
	}
	globalPrometheusExporter = promExporter

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String(serviceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating resource: %w", err)
	}

	var metricReaders []sdkmetric.Reader
	metricReaders = append(metricReaders, promExporter)

	var shutdownTracer func(context.Context) error
	var tp *sdktrace.TracerProvider

	if otelEndpoint != "" {
		// 2. Initialize OTLP gRPC tracing exporter
		traceExporter, err := otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(otelEndpoint),
			otlptracegrpc.WithInsecure(),
		)
		if err != nil {
			return nil, fmt.Errorf("creating OTLP trace exporter: %w", err)
		}

		tp = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(traceExporter),
			sdktrace.WithResource(res),
		)
		otel.SetTracerProvider(tp)
		shutdownTracer = tp.Shutdown

		// 3. Initialize OTLP gRPC metrics push exporter
		metricExporter, err := otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithEndpoint(otelEndpoint),
			otlpmetricgrpc.WithInsecure(),
		)
		if err != nil {
			return nil, fmt.Errorf("creating OTLP metric exporter: %w", err)
		}

		pushReader := sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(15*time.Second))
		metricReaders = append(metricReaders, pushReader)
	} else {
		// No-op tracing provider
		tp = sdktrace.NewTracerProvider(sdktrace.WithResource(res))
		otel.SetTracerProvider(tp)
		shutdownTracer = tp.Shutdown
	}

	// 4. Initialize unified MeterProvider with registered readers
	var opts []sdkmetric.Option
	opts = append(opts, sdkmetric.WithResource(res))
	for _, reader := range metricReaders {
		opts = append(opts, sdkmetric.WithReader(reader))
	}
	mp := sdkmetric.NewMeterProvider(opts...)
	otel.SetMeterProvider(mp)
	initInstruments()

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Combined shutdown function
	return func(shutdownCtx context.Context) error {
		// Enforce a short timeout for the shutdown process to avoid blocking component exit
		ctxWithTimeout, cancel := context.WithTimeout(shutdownCtx, 2*time.Second)
		defer cancel()

		if err := shutdownTracer(ctxWithTimeout); err != nil {
			otel.Handle(fmt.Errorf("failed to shutdown trace provider: %w", err))
		}
		if err := mp.Shutdown(ctxWithTimeout); err != nil {
			otel.Handle(fmt.Errorf("failed to shutdown meter provider: %w", err))
		}
		return nil
	}, nil
}

// Tracer returns a named tracer from the global provider.
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// TracingMiddleware returns a Gin middleware that extracts OTel trace context from incoming HTTP headers
// and sets it in the request context, starting a server span.
func TracingMiddleware(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		// Check if request path matches standard health probes and metrics
		if path == "/healthz" || path == "/readyz" || path == "/metrics" {
			// Extract trace context but bypass server span generation for system checks to avoid spam
			propagator := otel.GetTextMapPropagator()
			ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))
			c.Request = c.Request.WithContext(ctx)
			c.Next()
			return
		}

		// Extract incoming trace parent context using global TextMapPropagator
		propagator := otel.GetTextMapPropagator()
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		// Start a new server span using extracted context
		tracer := otel.Tracer(serviceName)
		ctx, span := tracer.Start(ctx, c.Request.Method+" "+path, trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()

		// Pass trace context down to Gin request context
		c.Request = c.Request.WithContext(ctx)

		c.Next()

		// Emit structured error events for 4xx and 5xx via the framework helper.
		// This covers all handlers uniformly without any per-handler wiring.
		afterRequest(c)
	}
}
