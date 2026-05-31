# TASK-M5.10.2: Cognitive Loop Engine Application Service

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.10.2                                |
| Status        | TODO                                        |
| Spec          | SPEC-FR-M5.10                               |
| Depends On    | TASK-M5.10.1                                |

## Description

Implement the iterative active reasoning loop (Thought-Action-Observation) inside `internal/agent/application/service/cognitive_engine.go`. Manage the ephemeral thought-loop state and coordinate iterative completions against the `Brain` outbound port.

## Work Items

1. **RED Phase**:
   - Write unit tests in `internal/agent/application/service/cognitive_engine_test.go` using mock `Brain` clients.
   - Assert that the engine halts reasoning and returns a final answer when the mock LLM outputs text instead of a tool call.
   - Assert that the engine halts reasoning, logs a warning, and returns the best available text response if loop iteration exceeds `TS_AGENT_MAX_REASONING_STEPS` (mocked as 3 in tests).
   - Run the tests and assert failure (RED).

2. **GREEN Phase**:
   - Create `internal/agent/application/service/cognitive_engine.go` implementing the ReAct loop logic.
   - Inject the `Brain` outbound port interface into the service constructor.
   - Manage loop state constraints (step counter, active thread history context, tool schema updates).
   - Run tests to verify compilation and execution (GREEN).

3. **REFACTOR Phase**:
   - Clean up loop termination conditions and context string construction helper methods.
   - Verify zero leakage of concrete client packages inside the application service.

## Acceptance Criteria

1. The cognitive engine executes multi-turn reasoning cycles, interacting exclusively with the stateless outbound `Brain` port interface.
2. Step limit bounds (`TS_AGENT_MAX_REASONING_STEPS`) are strictly enforced, ensuring loop termination and graceful warnings.
