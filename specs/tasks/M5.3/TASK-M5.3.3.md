# TASK-M5.3.3: Inbound Message Pipeline and Use Case Integration

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.3.3                                 |
| Status        | VERIFIED                                    |
| Spec          | SPEC-FR-M5.3                                |
| Depends On    | TASK-M5.3.2                                 |

## Description

Integrate the short-term memory (STM) port into the active inbound message processing pipeline. Extend the driving `MessageProcessor` port interface, modify `MessageProcessorService` to orchestrate the double-append reasoning loop, parse configurable limits (`TS_AGENT_STM_LIMIT`), handle connection failures gracefully with a stateless fallback, and update `EchoSubscriber` to capture and propagate the `thread_id` and tenant context.

## Work Items

1. **RED Phase**:
   - Update `internal/agent/application/service/message_processor_test.go` to assert:
     - The user turn is written to memory before calling the brain.
     - Conversational history is fetched and passed in the `BrainRequest`.
     - The brain response is written to memory after execution.
     - Graceful degradation: simulating Redis connection errors does not fail the use case and correctly falls back to stateless processing with a Zerolog warning.
   - Update `internal/agent/adapters/inbound/nats/echo_subscriber_test.go` to mock the updated processor interface and verify the thread ID and tenant ID propagate properly.
   - Run tests and witness failure (RED).

2. **GREEN Phase**:
   - Update `internal/agent/application/ports/inbound/message.go` to pass `tenantID`, `agentID`, and `threadID` along with `payload`.
   - Modify `MessageProcessorService` inside `internal/agent/application/service/message_processor.go`:
     - Accept `ShortTermMemory` outbound port in constructor.
     - Parse sliding window limit (`TS_AGENT_STM_LIMIT`, default: `10`) using Viper.
     - Implement the pipeline: Append User Turn -> Fetch sliding window history (with robust `try-catch` Redis failure logging and fallback logic) -> Generate response using Brain -> Append Assistant Turn -> Return.
   - Update `internal/agent/adapters/inbound/nats/echo_subscriber.go` to extract the `thread_id` from the inbound payload and invoke the extended processor.
   - Run tests and verify they pass (GREEN).

3. **REFACTOR Phase**:
   - Clean up mock structures and verify structured JSON logging attributes (`trace_id`, `span_id`, `tenant_id`, `thread_id`) correlate seamlessly.

## Acceptance Criteria

1. Inbound processing pipeline successfully compiles and propagates tenant, agent, and thread contexts across service boundaries.
2. Graceful fallback is active: Redis downtime triggers structured warnings but reasoning is completed statelessly without client-facing errors.
3. The LLM receives the proper history sequence (user and assistant turns in chronologically sorted order).
