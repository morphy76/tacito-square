# SPEC-FR-M2.5: Operator Hello World

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M2.5                                |
| Status        | DRAFT                                       |
| Milestone     | M2                                          |
| Component     | operator                                    |
| Depends On    | SPEC-FR-M2.2                                |
| Supersedes    | none                                        |

## Context

The operator watches Kubernetes CRDs and manages agent pod lifecycle. Before implementing reconciliation logic (M4), it needs a working Kubebuilder-based controller manager with health probes, structured logging, and tracing. The operator is conditionally deployed (`operator.enabled`).

## Specification

1. The `cmd/operator/main.go` entry point MUST initialize:
   - Viper configuration with `TS_OPERATOR_*` environment variable prefix.
   - zerolog logger with JSON output.
   - OpenTelemetry tracer with OTLP exporter.
   - controller-runtime Manager with health and readiness endpoints.
2. The Manager MUST register:
   - Health endpoint at `/healthz` (liveness).
   - Readiness endpoint at `/readyz`.
3. The server MUST listen on the port specified by `TS_OPERATOR_PORT` (default: 8082).
4. The Manager MUST handle graceful shutdown on SIGTERM/SIGINT.
5. Startup MUST log component name (`operator`), version (from `VERSION.operator`), and leader election status.

## Acceptance Criteria

1. `go build ./cmd/operator` produces an `operator` binary.
2. Running the binary starts the controller manager.
3. `/healthz` returns 200.
4. `/readyz` returns 200.
5. Stdout contains valid JSON log lines with `component: "operator"`.
6. SIGTERM triggers graceful shutdown.

## Test Plan

1. **Build**: `go build -o bin/operator ./cmd/operator` — exit code 0.
2. **Health**: Use envtest to start manager, verify `/healthz` → 200.
3. **Logging**: Capture stdout, parse JSON, assert fields.
4. **Shutdown**: Send SIGTERM, assert clean exit.

## Files Affected

- `cmd/operator/main.go` (MODIFY — bootstrap controller-runtime manager)
- `internal/operator/` (NEW — operator-specific bootstrap)
