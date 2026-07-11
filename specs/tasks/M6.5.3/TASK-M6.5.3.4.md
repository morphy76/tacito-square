# TASK-M6.5.3.4: Application Layer — Prompt & Agent Service Enhancements

| Field       | Value |
|-------------|-------|
| ID          | TASK-M6.5.3.4 |
| Status      | DRAFT |
| Spec        | SPEC-FR-M6.5.3 |
| Depends On  | TASK-M6.5.3.2, TASK-M6.5.3.3 |

## Description

Enhance application services `PromptService` and `AgentService` to orchestrate new CRUD options, versions, and agent associations.

## Work Items

1. **RED Phase**:
   - In `internal/keeper/application/service/prompt_system_test.go` or `agent_service_test.go`, write unit tests:
     - Verify `AgentService` exposes methods to attach and detach prompts/collections.
     - Verify `AgentService` exposes a method to retrieve the resolved prompt list.
     - Verify version number is incremented on prompt update.

2. **GREEN Phase**:
   - **Inbound Ports**:
     - Update `PromptUseCase` and `AgentUseCase` in `internal/keeper/application/ports/inbound/usecases.go`:
       - Add collection modification methods.
       - Add agent prompt attachment/detachment and resolved prompt list retrieval methods.
   - **Prompt Service**:
     - Update `internal/keeper/application/service/prompt_service.go` to handle collection membership modification and versions.
   - **Agent Service**:
     - Update `internal/keeper/application/service/agent_service.go` to handle prompt and collection attachment/detachment.
     - Implement resolved prompt list retrieval by invoking the `ResolveEffectivePrompts` domain service.

3. **REFACTOR Phase**:
   - Verify proper error handling/wrapping in use cases.
   - Run tests: `go test ./internal/keeper/application/service/...`.

## Acceptance Criteria

1. Inbound port interfaces match the new CRUD and association capabilities.
2. `AgentService` correctly delegates resolved prompt list compilation to the domain service.
3. Service layer unit tests compile and pass.
