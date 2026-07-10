# SPEC-FR-M6.5.8: Dynamic System Prompt Construction Pipeline

| Field       | Value                                    |
|-------------|------------------------------------------|
| ID          | SPEC-FR-M6.5.8                           |
| Status      | DRAFT                                    |
| Milestone   | M6.5                                     |
| Component   | agent                                    |
| Depends On  | SPEC-FR-M6.5.6, SPEC-FR-M6.5.7          |
| Supersedes  | none                                     |

## Context

The cognitive engine (`internal/agent/application/service/cognitive_engine.go`) must assemble
the final system prompt delivered to the LLM on every reasoning step. The current implementation
partially constructs this through `compileDynamicSystemPrompt()` but the exact layer boundaries,
persistence semantics, and role-based tool filtering are not formally specified.

With the delivery of the role-driven structural template (SPEC-FR-M6.5.7) and the
`PropagatedAgentConfig` shared schema (SPEC-FR-M6.5.6), the cognitive engine now has three
well-defined sources of system prompt content. This spec formalises the three-layer composition
model, the tool-exposure rules per role, and the persistence semantics for skill activation across
reasoning steps and thread turns.

## Specification

### 1. Three-Layer System Prompt Model

The cognitive engine assembles the system prompt in three ordered layers on each reasoning step.
The layers are concatenated with a visible delimiter line (`---`) between each:

```
[Layer 1 — Structural Template]
---
[Layer 2 — Persona & Instructions]
---
[Layer 3 — Activated Skills]   <- present only when at least one skill is active
```

#### Layer 1: Structural Template (static per agent session)

- **Source:** `/etc/tacito/system-prompt-template.txt` (rendered by Operator from Helm
  ConfigMap, per SPEC-FR-M6.5.7).
- **Loaded:** once at agent startup; stored in-memory as `baseSystemPrompt`.
- **Content:** the role posture instructions — who the agent is, what its primary cognitive mode
  is (delegate vs. execute), and how to use its built-in tools.
- **Mutability:** immutable within a single agent session. It does not change between threads or
  turns.

#### Layer 2: Persona & Instructions (static per agent session)

- **Source:** `PropagatedAgentConfig.Directives` (concatenated active prompt contents, assembled
  by Keeper per SPEC-FR-M6.5.6).
- **Loaded:** once at agent startup from `/etc/tacito/agent-config.json`; stored in-memory as
  `personaDirectives`.
- **Content:** persona definition, behavioural constraints, community-specific instructions
  assembled from the resolved prompt collection.
- **Mutability:** immutable within a single agent session.

#### Layer 3: Activated Skills (dynamic per reasoning step)

- **Source:** skills enabled via `enable_skill` tool calls during the current reasoning loop,
  plus skills persisted in the STM thread history from previous turns.
- **Content:** each enabled skill's `SkillConfig.Content` field, prefixed with a skill header:
  ```
  ## Skill: {skill_name}
  {skill_content}
  ```
- **Mutability:** grows during a reasoning loop as the LLM enables more skills. Persisted across
  thread turns by recording enabled skill names in STM history metadata.

### 2. `compileDynamicSystemPrompt` Contract

```go
// compileDynamicSystemPrompt assembles the full system prompt for the current reasoning step.
// activatedSkills is the set of skill names enabled so far in the current turn (may be empty).
func (e *CognitiveEngine) compileDynamicSystemPrompt(activatedSkills []string) string
```

Implementation rules:
1. Always start with `e.baseSystemPrompt` (Layer 1).
2. Append delimiter `\n---\n` then `e.personaDirectives` (Layer 2).
3. If `activatedSkills` is non-empty:
   a. Append delimiter `\n---\n`.
   b. For each name in `activatedSkills`, look up the matching `SkillConfig` from
      `e.availableSkills` (loaded from `PropagatedAgentConfig.Skills`).
   c. Append the skill header and content.
4. Return the assembled string.

### 3. Tool Presentation Per Reasoning Step

At the start of each reasoning step the engine builds the `[]ToolDefinition` slice exposed to the
LLM. The set of tools depends on the agent's role:

#### Built-in tools (all roles):
- `enable_skill`: function-call tool with an enum of available skill names drawn from
  `PropagatedAgentConfig.Skills[*].Name`. Present only when `len(PropagatedAgentConfig.Skills) > 0`.
- `read_large_payload` / `write_large_payload`: always present.
- `recall_memory`: present only when LTM is enabled in configuration.

#### MCP tools (non-hub roles only):
- Sourced from `mcpExecutor.ListAllowedTools()`.
- Added to `[]ToolDefinition` only when the agent's role is `standalone` or `spoke`.
- For a **hub** agent, `mcpExecutor.ListAllowedTools()` is not called and no MCP tools are
  exposed to the LLM.

#### Hub-only built-in tools:
- `list_available_agents`: returns the current non-stale spoke agent cards (see
  SPEC-FR-M6.5.13).
- `delegate_to_agent`: delegates a task to a specific spoke (existing mechanism from
  SPEC-FR-M6.6).

All built-in and MCP tools appear in the **same** `[]ToolDefinition` slice passed in
`BrainRequest.Tools`. There is no separate list.

### 4. `enable_skill` Execution Semantics

When the LLM issues an `enable_skill` function call with argument `name`:

1. Look up the `SkillConfig` with matching `Name` in `PropagatedAgentConfig.Skills`. If not
   found, return a tool-error observation: `"skill not found: {name}"`.
2. Add `name` to `activatedSkills` for the current turn.
3. Return the skill's `Content` as the tool observation so the LLM receives it immediately.
4. On the next reasoning step, `compileDynamicSystemPrompt(activatedSkills)` will include the
   skill content in Layer 3.
5. Record the enabled skill name in the STM history entry for the current thread turn so it can
   be re-injected on subsequent turns (see §5).

### 5. Skill Persistence Across Thread Turns

At the **start** of each new turn (thread step) the engine:
1. Reads the STM history for the current thread.
2. Extracts previously enabled skill names from history metadata (stored as a JSON array under
   metadata key `"enabled_skills"`).
3. Initialises `activatedSkills` with these names before presenting any tools to the LLM.
4. The LLM thus "sees" the previously enabled skills in Layer 3 without having to re-call
   `enable_skill`.

This reconstructs the context the LLM had at the end of the previous turn without replaying tool
calls.

### 6. MCP Tool Execution Semantics

When the LLM calls an MCP tool (by a name returned by `mcpExecutor.ListAllowedTools()`):

1. Dispatch the call to the external MCP server via `mcpExecutor.Execute(ctx, toolName, args)`.
2. Return the MCP response as the tool observation.
3. No system prompt modification occurs (MCP tool calls do not modify Layer 3).
4. Errors from the MCP server are returned as structured tool-error observations; they do not
   cause the reasoning loop to abort unless the error is unrecoverable.

## Acceptance Criteria

1. On the first reasoning step of a new thread, the system prompt sent to the LLM contains
   Layer 1 (`baseSystemPrompt`) and Layer 2 (`personaDirectives`) only, with the `---`
   delimiter between them. Layer 3 is absent.
2. After the LLM calls `enable_skill("routing_policy")`, the next reasoning step's system
   prompt contains Layer 3 with the `## Skill: routing_policy` header and the skill content.
3. On the first step of a second turn in the same thread, previously enabled skills are
   re-injected into Layer 3 without the LLM re-calling `enable_skill`.
4. A hub agent's `[]ToolDefinition` does not contain any MCP tool. It does contain
   `list_available_agents` and `delegate_to_agent`.
5. A standalone/spoke agent's `[]ToolDefinition` contains both MCP tools and built-in tools
   (`enable_skill`, `read_large_payload`, etc.) in the same slice.
6. Calling `enable_skill` with a name not present in `PropagatedAgentConfig.Skills` returns a
   tool-error observation and does not modify `activatedSkills`.
7. All existing tests in `cognitive_engine_test.go` pass without modification.

## Test Plan

### Unit — `compileDynamicSystemPrompt`
- With empty `activatedSkills`: assert output contains Layer 1 and Layer 2 separated by `---`,
  no Layer 3 content present.
- With `activatedSkills = ["skill_a"]`: assert output contains Layer 3 with `## Skill: skill_a`
  header and the correct content.
- With two activated skills: assert both appear in Layer 3 in order.

### Unit — Skill persistence
- Simulate a thread history with `"enabled_skills": ["routing_policy"]` in STM metadata.
- Assert that at the start of the next turn, `activatedSkills` is pre-populated with
  `["routing_policy"]` before any tool calls.

### Unit — Role-based tool filtering
- Instantiate a hub-role cognitive engine; assert `buildToolDefinitions()` returns no MCP tools
  and does include `list_available_agents`.
- Instantiate a spoke-role cognitive engine; assert `buildToolDefinitions()` includes MCP tools
  and does not include `list_available_agents`.

### Unit — `enable_skill` error path
- Call `enable_skill` with a name absent from `PropagatedAgentConfig.Skills`; assert the
  returned observation is a tool-error string and `activatedSkills` is unchanged.

### Regression
- Run the full `cognitive_engine_test.go` suite and assert all existing tests pass.

## Files Affected

| File | Change |
|------|--------|
| `internal/agent/application/service/cognitive_engine.go` | **MODIFY** — formalise `compileDynamicSystemPrompt` three-layer model; role-based tool slice assembly; skill persistence read/write |
| `internal/agent/application/service/cognitive_engine_types.go` | **MODIFY** — use `agentconfig.PropagatedAgentConfig` from shared package; remove duplicate local struct |
| `internal/agent/application/service/cognitive_engine_test.go` | **MODIFY** — add pipeline unit tests (layers, filtering, persistence); maintain all existing assertions |
