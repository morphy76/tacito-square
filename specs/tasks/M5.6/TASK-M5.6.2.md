# TASK-M5.6.2: Agent Cognitive Tool read_large_payload

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.6.2                                 |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M5.6                                |
| Depends On    | TASK-M5.6.1                                 |

## Description

Implement the built-in cognitive tool `read_large_payload` in the agent's cognitive reasoning engine. This tool allows the LLM reasoning loop to dynamically retrieve large contents stored in S3/MinIO when it identifies an `s3_reference` block in the conversation thread. To prevent pod memory exhaustion and comply with strict Kubernetes resource boundaries, all read operations MUST be actively streamed and processed via buffered structures (e.g., `bufio.Reader` or chunked reader channels) rather than loading the entire object into memory at once.

## Work Items

1. **RED Phase**:
   - Write a unit/integration test in `internal/agent/application/service/cognitive_engine_test.go` asserting that:
     - The tool list sent to the LLM includes the `read_large_payload` tool.
     - When the reasoning engine executes `read_large_payload` with a specific S3 key, it queries the mock `BlobStore` using streamed/buffered mechanisms and returns the file content as the observation string.
     - Memory profiling assertions verify that large reads do not cause flat in-memory block expansion.
   - Run tests and verify failure (RED).

2. **GREEN Phase**:
   - Define the `read_large_payload` tool schema in `internal/agent/domain/model/reasoning.go`.
   - Register the `read_large_payload` tool execution inside `internal/agent/application/service/cognitive_engine.go`.
   - Implement the S3 `Get` execution within the tool execution using streamed and buffered reads, returning the text content to the LLM.
   - Run tests and verify successful completion (GREEN).

3. **REFACTOR Phase**:
   - Wrap the S3 `Get` invocation inside the circuit breaker.
   - Ensure optimal buffer chunk sizing is used to prevent memory limits aggression.
   - Implement a graceful fallback: if S3 is down or the circuit breaker is open, the tool returns `{"error": "Object storage temporarily unavailable."}` to the LLM instead of failing the reasoning turn.

## Acceptance Criteria

1. The `read_large_payload` tool is correctly registered and visible to the LLM.
2. Invoking the tool correctly fetches content from S3 and exposes it as a text observation in the reasoning trace.
3. No flat in-memory slice loading of the downloaded payload occurs; retrieval is fully stream-buffered.
4. Outage simulation results in a graceful JSON error observation without causing reasoning loop crashes.
