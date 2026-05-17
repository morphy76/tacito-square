# SPEC-FR-09.1: OpenTelemetry Tracing

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-FR-09.1                       |
| Status        | VERIFIED                           |
| Milestone     | M1                                 |
| FR/NFR Ref    | FR-09.1                            |
| Component     | shared                             |
| Depends On    | —                                  |

## Context

Distributed tracing is essential for observability across agent-keeper-infrastructure boundaries. OTel with OTLP/gRPC is the standard.

## Specification

1. `InitTracer(ctx, serviceName, serviceVersion, otelEndpoint)` MUST:
   a. Create an OTLP gRPC exporter pointing to the configured endpoint
   b. Create a `TracerProvider` with service resource attributes (`service.name`, `service.version`)
   c. Set global `TracerProvider` and `TextMapPropagator` (W3C TraceContext + Baggage)
   d. Return a shutdown function for graceful teardown
2. If `otelEndpoint` is empty, MUST return a no-op shutdown (tracing disabled gracefully).
3. `Tracer(name)` MUST return a named tracer from the global provider.
4. Connection MUST be insecure for dev (OTLP gRPC without TLS).

## Acceptance Criteria

1. InitTracer with valid endpoint registers global TracerProvider
2. InitTracer with empty endpoint returns no-op shutdown
3. Tracer returns a usable tracer instance
4. W3C TraceContext propagation is set globally

## Files

- `internal/shared/observability/tracing.go` ✅ IMPLEMENTED
