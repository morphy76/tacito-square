# TASK-M6.1.4: Agent Asynchronous Orchestration State Machine & Discovery

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M6.1.4                                 |
| Status        | IN_PROGRESS                                 |
| Spec          | SPEC-FR-M6.1                                |
| Depends On    | TASK-M6.1.1, TASK-M6.1.3                    |

## Description

Design and implement the Hub agent's orchestration state machine inside the cognitive engine. The Hub must read Spoke agent profile capability data ("Agent Cards") to match tasks, execute routing turns asynchronously by saving states in Redis and yielding execution, and coordinate replica concurrency via Redis distributed locks.

## Boundary & Target Functions

- **Package**: `internal/agent/application/service`, `internal/agent/adapters/outbound/redis`
- **Files**:
  - `internal/agent/application/service/cognitive_engine.go`
  - `internal/agent/application/service/schema_router_impl.go`
  - `internal/agent/adapters/outbound/redis/memory_adapter.go` (or a dedicated lock repository)

## Work Items

1. **RED Phase (Write Tests First)**:
   * Write unit tests for the Hub's asynchronous routing loop:
     * Mock the Agent Cards store to return mock Spokes and capabilities.
     * Assert that sending a user turn triggers a semantic routing decision.
     * Verify the Hub saves the active state to Redis and returns a yield signal (yielding instead of blocking).
     * Verify that when the Spoke response event arrives, the Hub loads the state, proceeds to the next state, or finalizes.
     * Assert that distributed lock operations are invoked on Redis when starting and finishing thread processing.

2. **GREEN Phase (Implement Minimum Code)**:
   * Implement semantic routing logic in the cognitive engine utilizing Agent Cards.
   * Add Redis state serialization methods to store and resume orchestration states for a thread.
   * Implement the asynchronous state machine event handlers:
     * Handler for user message event: lock thread -> load state/history -> choose spoke -> publish task to spoke -> save state -> unlock thread -> exit.
     * Handler for spoke response event: lock thread -> load state -> process response -> determine next action (finish or route to another spoke) -> publish response/task -> unlock thread -> exit.
   * Add simple distributed lock helpers using Redis key/value primitives.

3. **REFACTOR Phase (Clean & Generalize)**:
   * Clean up state representation.
   * Ensure trace context propagation persists across asynchronous yields.

## Acceptance Criteria

1. Asynchronous routing and orchestration tests pass successfully.
2. The Hub uses Redis to store and resume conversation state across turns.
3. Thread locks prevent concurrent processing of duplicate events across replica instances.
