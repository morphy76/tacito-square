# TASK-M5.6.4: Agent Cognitive Tool write_large_payload

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.6.4                                 |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M5.6                                |
| Depends On    | TASK-M5.6.2                                 |

## Description

Implement the built-in cognitive tool `write_large_payload` in the agent's cognitive reasoning engine. This tool allows the LLM reasoning loop to dynamically offload generated, mutated, or large text payloads to S3/MinIO. To ensure compliance with strict Kubernetes container resource boundaries and avoid pod memory limit aggression, all write operations MUST be actively streamed and buffered (e.g., using `io.Reader` and chunked stream buffers) instead of loading the entire content into a flat memory slice at once.

## Work Items

1. **RED Phase**:
   - Write a unit/integration test in `internal/agent/application/service/cognitive_engine_test.go` asserting that:
     - The tool list sent to the LLM includes `write_large_payload`.
     - When the reasoning engine executes `write_large_payload` with text payload parameters, it invokes the mock S3 `BlobStore` using streamed `io.Reader` interfaces and returns the structured S3 reference block.
     - Memory allocation profile assertions verify that large payloads do not cause flat slice memory expansion.
   - Run tests and verify failure (RED).

2. **GREEN Phase**:
   - Define the `write_large_payload` tool schema in `internal/agent/domain/model/reasoning.go`.
   - Register the `write_large_payload` tool execution inside `internal/agent/application/service/cognitive_engine.go`.
   - Implement S3 streamed `Put` execution using the shared `BlobStore` port interface, returning the structured S3 reference block (with normalized bucket name and isolation keys) as the observation.
   - Run tests and verify successful completion (GREEN).

3. **REFACTOR Phase**:
   - Wrap the S3 `Put` invocation inside the circuit breaker.
   - Ensure that the stream buffer sizes (e.g., in chunked readers) are optimized for Kubernetes memory ceilings.
   - Implement standard fallback logic: if S3 is down, return the graceful error block `{"error": "Object storage temporarily unavailable."}` to the LLM reasoning loop.

## Acceptance Criteria

1. The `write_large_payload` tool is correctly registered and visible to the LLM.
2. Invoking the tool results in a streamed, chunked upload to S3, returning a valid S3 reference JSON block as the observation.
3. No flat in-memory slice loading of the target payload occurs during the S3 adapter `Put` execution.
