# TASK-M5.4.3: LLM Embeddings Port Adapter and Resiliency Circuit Breaker

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.4.3                                 |
| Status        | VERIFIED                                    |
| Spec          | SPEC-FR-M5.4                                |
| Depends On    | TASK-M5.4.1                                 |

## Description

Implement the outbound `Embedder` interface in the OpenAI adapter to generate dense vector embeddings using the configured brain embedding model. Wrap both Qdrant and Embedder outbound network operations in independent circuit breakers and exponential backoff retry wrappers.

## Work Items

1. **RED Phase**:
   - Write unit and integration tests inside `internal/agent/adapters/outbound/openai/openai_adapter_test.go` asserting:
     - Embedding generation correctly converts plain text to standard float vector representations.
     - Outbound calls are protected by circuit breakers (asserting fallback execution or error suppression when thresholds are breached).
   - Run tests and witness failure (RED).

2. **GREEN Phase**:
   - Update `internal/agent/adapters/outbound/openai/openai_adapter.go` to implement the `Embedder` outbound port.
   - Wire embedding generation calls using the official `client.Embeddings.New` API endpoints.
   - Register independent `resiliency.CircuitBreaker` instances for Qdrant and Embedder outbound call blocks.
   - Run the tests to verify compilation and successful execution (GREEN).

3. **REFACTOR Phase**:
   - Clean up model configurations and ensure explicit deadlines are always propagated inside the embedding retrieval `context.Context`.

## Acceptance Criteria

1. The OpenAI adapter successfully generates high-dimensional embeddings.
2. Embedding operations are fully governed by exponential backoff retry loops and protected by tested circuit breakers.
