# TASK-M5.6.3: Agent Main Wiring & Parallel Health Probes

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.6.3                                 |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M5.6                                |
| Depends On    | TASK-M5.6.4                                 |

## Description

Wire up S3 and MinIO connection parameters in the agent entry point (`cmd/agent/main.go`), instantiate the shared S3 `BlobStoreAdapter` using the configured parameters, and inject it as a dependency into the NATS `echo_subscriber` and `MessageProcessor`.
Additionally, support S3 opt-out/enabled configuration via the environment variable `TS_AGENT_S3_ENABLED` (mapped via Viper as `s3.enabled`, defaulting to `false`). If S3 is disabled, S3 adapters and checkers MUST NOT be initialized, and `nil` must be passed to the subscriber and reasoning loop, automatically opting out of the capability.
Finally, implement parallel connection check integration within the `/readyz` health checker probe if S3 is enabled.

## Work Items

1. **RED Phase**:
   - Write an integration/unit test in `internal/agent/bootstrap_test.go` or `cmd/agent/main_test.go` verifying that:
     - When `TS_AGENT_S3_ENABLED` is `false`, the `"minio"` health check is NOT registered, and no S3 clients are created.
     - When `TS_AGENT_S3_ENABLED` is `true` and the S3/MinIO connection is offline, the `/readyz` endpoint returns a `503 Service Unavailable` status code with the `"minio"` key details populated.
   - Run tests and verify failure (RED).

2. **GREEN Phase**:
   - Update `cmd/agent/main.go` to parse S3 environment variables (`TS_AGENT_S3_ENABLED`, `TS_AGENT_S3_ENDPOINT`, `TS_AGENT_S3_BUCKET`, `TS_AGENT_S3_OFFLOAD_THRESHOLD`, `TS_AGENT_S3_MAX_READ_SIZE`, `TS_AGENT_S3_CHUNK_SIZE`).
   - Map `TS_AGENT_S3_ENABLED` defaulting to `false` via Viper.
   - Conditionally instantiate the shared S3 concrete adapter `s3.NewBlobStoreAdapter` only if `s3.enabled` is `true`.
   - Wire this adapter into the NATS subscriber constructor and the message processing bootstrap stack. If disabled, pass `nil` (retaining the robust graceful bypasses).
   - Conditionally add the custom `health.Checker` under the name `"minio"` in the checkers list only if S3 is enabled.
   - Run tests and verify successful completion (GREEN).

3. **REFACTOR Phase**:
   - Ensure the S3 client uses the centralized instrumented HTTP client to correctly inject tracing spans and handle network timeouts.
   - Ensure any graceful shutdown procedures safely release S3 adapter connection references if enabled.

## Acceptance Criteria

1. Running `make build-agent` succeeds without compilation errors.
2. S3/MinIO is completely opted out by default; starting the agent without configuring S3 does not cause startup failures or register health checks.
3. Setting `TS_AGENT_S3_ENABLED=true` successfully registers and triggers the parallel `/readyz` health check for MinIO.
