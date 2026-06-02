# TASK-M5.6.3: Agent Main Wiring & Parallel Health Probes

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.6.3                                 |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M5.6                                |
| Depends On    | TASK-M5.6.2                                 |

## Description

Wire up S3 and MinIO connection parameters in the agent entry point (`cmd/agent/main.go`), instantiate the shared S3 `BlobStoreAdapter` using the configured parameters, and inject it as a dependency into the NATS `echo_subscriber` and `MessageProcessor`. Additionally, implement parallel connection check integration within the `/readyz` health checker probe.

## Work Items

1. **RED Phase**:
   - Write a unit/integration test in `internal/agent/bootstrap_test.go` verifying that:
     - A dependency health checker named `"minio"` is registered inside the health probe.
     - When the S3/MinIO connection is offline (simulated by making the checker fail), the `/readyz` endpoint returns a `503 Service Unavailable` status code with the `"minio"` key details populated.
   - Run tests and verify failure (RED).

2. **GREEN Phase**:
   - Update `cmd/agent/main.go` to parse S3 environment variables (`TS_AGENT_S3_ENDPOINT`, `TS_AGENT_S3_BUCKET`, `TS_AGENT_S3_OFFLOAD_THRESHOLD`).
   - Instantiate the shared S3 concrete adapter `s3.NewBlobStoreAdapter`.
   - Wire this adapter into the NATS subscriber constructor and the message processing bootstrap stack.
   - Add a custom `health.Checker` under the name `"minio"` in `internal/agent/bootstrap.go` or `cmd/agent/main.go` that pings/verifies S3 availability.
   - Run tests and verify successful completion (GREEN).

3. **REFACTOR Phase**:
   - Ensure the S3 client uses the centralized instrumented HTTP client to correctly inject tracing spans and handle network timeouts.
   - Ensure any graceful shutdown procedures safely release S3 adapter connection references.

## Acceptance Criteria

1. Running `make build-agent` succeeds without compilation errors.
2. The `/readyz` endpoint pings MinIO in parallel and correctly reflects its health status.
3. S3 configurations are mapped to environment variables and correctly override defaults.
