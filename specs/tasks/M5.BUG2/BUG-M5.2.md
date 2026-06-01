# BUG-M5.2: Skills Misinterpreted as Tool Collections Instead of Dynamic Procedural Knowledge Sources

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M5.2                                                           |
| Status        | OPEN                                                               |
| Severity      | HIGH                                                               |
| Milestone     | M5 — Agent Core                                                    |
| Affects       | `internal/agent/application/service/cognitive_engine.go`           |
| Violates      | SPEC-FR-M3.3, SPEC-FR-M5.10                                        |
| Discovered    | Architectural review and clarification of dynamic skills execution |

## Problem Statement

In the current implementation of Milestone 5 (`CognitiveEngine` reasoning loop), "skills" have been interpreted and implemented as collections of executable tools (specifically in `RegisterSkillCollection` and the `enable_skill` tool handler). This implementation conflates skills with MCP tools and contradicts their conceptual definition.

Per the correct architectural design:
1. **MCP Tools** are interactive I/O capabilities provided via Model Context Protocol (MCP) server bindings.
2. **Skills** are procedural knowledge sources (consisting of a name, description, and full procedural prompt/content) that dynamically enrich the LLM's reasoning context.
3. The reasoning loop should expose all available skills and their descriptions to the brain.
4. The brain should decide which skill is needed based on those descriptions, then dynamically load just the full procedural content of the needed skill(s) to enrich the context, rather than loading all skill contents at once.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| Agent Cognitive Engine | `internal/agent/application/service/cognitive_engine.go` | Implements `skillPool` as `map[string]map[string]ToolHandler` and registers skills as collections of tools. The `enable_skill` handler registers tools rather than loading procedural knowledge content. |
| Agent Cognitive Engine Tests | `internal/agent/application/service/dynamic_skills_test.go` | Asserts the incorrect dynamic tool registration behavior instead of dynamic procedural knowledge loading. |
| Agent Main Bootstrap | `cmd/agent/main.go` | Sets up and registers mock skill collections as tools. |

## Impact

1. **Incorrect conceptual model**: Treats skills as programmatic tools rather than specialized knowledge extensions, violating their design.
2. **Context inflation**: Without dynamic procedural knowledge loading, either all guidelines must be pre-loaded into the system prompt (inflating context window and costs) or they cannot be utilized dynamically by the reasoning brain.
3. **Integration mismatch**: Interferes with the upcoming MCP tool integration by overlapping tool execution pathways under different names.

## Expected Behaviour

1. The `CognitiveEngine` MUST represent a Skill as a procedural knowledge structure containing:
   - `Name`: Unique string identifier (e.g. `math`).
   - `Description`: High-level summary of when/why to use the skill.
   - `Content`: The full procedural guidelines/instructions to inject when loaded.
2. The `CognitiveEngine` MUST allow registering Skills via a clean registration method (e.g. `RegisterSkill(skill Skill)`).
3. The cognitive loop system prompt or history context MUST expose available skill names and descriptions to the brain at the start of execution so it can make an informed decision on which skill(s) are needed.
4. The `enable_skill` (or `load_skill`) tool MUST be registered to allow the brain to load a specific skill. When called, it MUST retrieve the full `Content` of that skill and return it as the tool observation to dynamically enrich the conversation/reasoning context for subsequent steps.
5. Only the content of the selected skill(s) should be loaded; other registered skills' contents must remain unloaded.

## Acceptance Criteria

1. **Procedural Knowledge Loading**:
   - The brain can see names and descriptions of registered skills.
   - When the brain invokes the skill-loading tool, the exact procedural instructions/guidelines of the selected skill are injected into the reasoning loop's trace context.
2. **Clean TDD Execution**:
   - Tests in `dynamic_skills_test.go` demonstrate a mock brain identifying a needed skill, loading it dynamically, and utilizing the newly loaded procedural content to answer a query.
   - All tests pass cleanly.
