# TASK-M5.2.2: LLM Adapters with Resiliency (OpenAI & Ollama)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.2.2                                 |
| Status        | VERIFIED                                    |
| Spec          | SPEC-FR-M5.2                                |
| Depends On    | TASK-M5.2.1                                 |

## Description

Implement concrete, driven adapters for OpenAI and Ollama LLM provider engines in the adapters layer, integrating configuration lookup (Viper), configurable timeouts, retries with jitter, and circuit breaker protection.

## Work Items

1. **RED Phase**:
   - Write unit and mock validation tests in `internal/agent/adapters/outbound/openai/openai_adapter_test.go` and `internal/agent/adapters/outbound/ollama/ollama_adapter_test.go` verifying:
     - Initialization default mappings when no environment variables are set.
     - Timeout propagation utilizing a custom test context.
     - Retry logic executing upon simulated transient HTTP failures (e.g. 503 errors).
     - Circuit breaker tripping when simulated error thresholds are crossed.
   - Run tests and witness expected failures (RED).

2. **GREEN Phase**:
   - Implement configuration reading logic for `TS_AGENT_BRAIN_*` and provider-specific environment variables.
   - Create `internal/agent/adapters/outbound/openai/openai_adapter.go` implementing the `Brain` interface using `openai-go`.
   - Create `internal/agent/adapters/outbound/ollama/ollama_adapter.go` implementing the `Brain` interface using a resilient HTTP client.
   - Integrate standard circuit breaker patterns (e.g., using a lightweight, stack-approved resiliency library or custom circuit breaker logic).
   - Ensure explicit request timeouts are enforced and propagated via context.
   - Verify that all unit and mock resilience tests pass successfully (GREEN).

3. **REFACTOR Phase**:
   - Refactor shared HTTP client and retry configurations.
   - Verify OpenTelemetry spans are correctly injected into outbound HTTP request headers.

## Acceptance Criteria

1. Both adapters implement the `Brain` port interface.
2. The system defaults to OpenAI when `TS_AGENT_BRAIN_PROVIDER` is empty or missing.
3. Timeout, retry, and circuit breaker mechanisms operate as specified without resource leaks.
