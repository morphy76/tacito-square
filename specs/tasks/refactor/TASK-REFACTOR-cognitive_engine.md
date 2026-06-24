# TASK-REFACTOR-cognitive_engine: Refactor cognitive_engine.go

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-REFACTOR-cognitive_engine              |
| Status        | VERIFIED                                    |
| Target File   | [internal/agent/application/service/cognitive_engine.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/application/service/cognitive_engine.go) |
| Baseline Tests| All existing tests MUST pass without changes |

## Description
Decompose the God file `cognitive_engine.go` (858 LOC) to improve maintainability, readability, and context efficiency. We will split context helpers, data structures, and built-in tool implementations into separate focused files inside the same package (`service`), while keeping the core reasoning loop orchestrator in `cognitive_engine.go`.

## Work Items
1. **Baseline Phase**:
   - Ensure all existing tests pass successfully.
2. **Refactor Phase**:
   - `[x]` Create `internal/agent/application/service/cognitive_engine_types.go` and extract the following:
     - Context keys and context getters: `tenantCtxKey`, `agentCtxKey`, `threadCtxKey`, `activeToolsKey`, `parsedSkillsKey`, `GetTenantID`, `GetAgentID`, `GetThreadID`.
     - Types/structs: `Skill`, `PropagatedAgentConfig`, `parsedResponse`, `toolCallDetail`.
   - `[x]` Create `internal/agent/application/service/cognitive_engine_tools.go` and extract the following built-in tool handler implementations:
     - `handleReadLargePayload`
     - `handleWriteLargePayload`
     - `normalizeBucketName`
     - `handleRecallMemory`
     - `handleEnableSkill`
   - `[x]` Simplify `cognitive_engine.go` by removing the extracted types, helper methods, and tool handlers, leaving only:
     - Struct declaration `CognitiveEngine` and constructors/builder options.
     - Core loop orchestration logic: `ExecuteReasoningLoop` and `executeStep`.
     - Logging and NATS publishing helpers: `logStep` and `emitStepEvent`.
   - `[x]` Document all extracted files with concise docstrings and optimize imports.
3. **Verification Phase**:
   - Run the existing test suite (`make test`) to ensure all tests remain 100% green without modifying a single test file.
   - Run the project linter (`make lint`) to confirm codebase styling compliance.

## Acceptance Criteria
1. No existing unit/integration/contract test is modified.
2. `cognitive_engine.go` is reduced in size by approximately 280 LOC.
3. Code structure is cleaner, highly readable, and conforms to cognitive nesting limits.
4. All tests and linters pass cleanly.
