# TASK-M5.10.4: Dynamic Skill Activation and Tool Routing

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.10.4                                |
| Status        | IMPLEMENTED                                 |
| Spec          | SPEC-FR-M5.10                               |
| Depends On    | TASK-M5.10.2                                |

## Description

Implement the `enable_skill` built-in cognitive tool, enabling the reasoning engine to dynamically register and expose authorized MCP skill collections during active reasoning loops.

## Work Items

1. **RED Phase**:
   - Write unit tests in `internal/agent/application/service/dynamic_skills_test.go` asserting dynamic tool schema injection.
   - Assert that invoking `enable_skill` with an authorized skill name adds all whitelisted MCP tools of that collection to the LLM's prompt parameters in the subsequent completion cycle.
   - Assert that requesting an unauthorized skill name returns `"Skill unauthorized or not found."` error observation.
   - Run tests and verify failure (RED).

2. **GREEN Phase**:
   - Register the `enable_skill` tool schema into the cognitive engine's base toolset.
   - Implement the `enable_skill` execution handler inside `cognitive_engine.go`, pulling MCP server and tool mappings from the agent configuration pool.
   - Update the active tool definitions parameter generated for subsequent loop iteration cycles.
   - Run the tests to verify green status (GREEN).

3. **REFACTOR Phase**:
   - Refactor tool schema translation mappings to prevent duplicate registrations and ensure clean formatting.

## Acceptance Criteria

1. The LLM starts with only base system tools (`recall_memory`, `enable_skill`) exposed.
2. Exposing additional MCP tools occurs strictly after the LLM completes an `enable_skill` action turn, preventing prompt bloat.
