# TASK-M5.4.5: Agent Bootstrap Wiring, /readyz Health Probe, and E2E Test Suite Verification

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.4.5                                 |
| Status        | VERIFIED                                    |
| Spec          | SPEC-FR-M5.4                                |
| Depends On    | TASK-M5.4.4                                 |

## Description

Perform the final bootstrap wiring for Qdrant client connections in the Agent startup flow. Update the parallel `/readyz` health probe to check Qdrant availability as required by `RULE[k8s-best-practices.md]`. Bind configuration variables inside Viper, and execute full end-to-end (E2E) verification tests to validate semantic retrieval.

## Work Items

1. **RED Phase**:
   - Write or update bootstrap integration tests (e.g. `internal/agent/bootstrap_test.go`) and health check tests to assert that `/readyz` fails when Qdrant is unreachable (while `/healthz` remains healthy).
   - Run tests and witness failure (RED).

2. **GREEN Phase**:
   - Update `internal/agent/bootstrap.go` to initialize the Qdrant gRPC connection pool using Viper configurations:
     - `TS_AGENT_QDRANT_URL` (Qdrant gRPC address, e.g. `localhost:6334`).
     - `TS_AGENT_QDRANT_COLLECTION_NAME` (Collection name, default `ts_agent_memories`).
     - `TS_AGENT_QDRANT_VECTOR_DIMENSION` (Embedding dimensions, default `1536`).
     - Wire `QdrantLTMAdapter` and `OpenAIEmbedderAdapter` (or extended `openai_adapter`) into the `MessageProcessorService` constructor.
   - Update `/readyz` health handler to execute parallel connectivity checks (Ping) to Qdrant.
   - Run tests and verify the complete bootstrap wiring and health probes pass successfully (GREEN).

3. **REFACTOR Phase**:
   - Clean up configuration properties and ensure connection pooling parameters comply with `RULE[cloud-first.md]`.

## Acceptance Criteria

1. Agent bootstrap successfully connects to Qdrant and registers all outbound services in the runtime pipeline.
2. The `/readyz` health probe includes Qdrant connectivity checks, responding with `503 Service Unavailable` on failures.
3. Full E2E reasoning flow operates with both active short-term and long-term semantic memories active.
