# BUG-M5.4: Agent Tooling Violates MCP-First Architecture due to Mock Tools Embedded in Core Engine

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M5.4                                                           |
| Status        | OPEN                                                               |
| Severity      | MEDIUM                                                             |
| Milestone     | M5 — Agent Core                                                    |
| Affects       | `internal/agent/application/service/cognitive_engine.go`           |
| Violates      | SPEC-FR-M5.5, SPEC-FR-M5.10                                        |
| Discovered    | Architectural alignment check of agent tooling capabilities         |

## Problem Statement

In the current implementation of Milestone 5, the agent's core `CognitiveEngine` bootstrap (`cmd/agent/main.go` and `cognitive_engine.go`) contains hardcoded local tool handler functions (such as `utility_ping`, `math_add`, and `restricted_access`).

This violates the decoupled, MCP-first architectural boundary:
1. **MCP Decoupling**: All interactive/programmatic tools (databases, shell tools, third-party API clients) MUST be managed and executed strictly through Model Context Protocol (MCP) server bindings rather than being hardcoded into the agent core binary.
2. **Core Pollution**: Registering local go function handlers directly in the cognitive engine pollutes the core agent reasoning layer with domain-specific utility logic, introducing strong compilation coupling.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| Agent Main Bootstrap | `cmd/agent/main.go` | Directly implements and registers local mock tools (`restricted_access`, `math_add`, `utility_ping`) into the cognitive engine. |
| Cognitive Engine | `internal/agent/application/service/cognitive_engine.go` | Exposes local registration primitives (`RegisterTool` and `ToolHandler`) that facilitate embedding local executable code. |

## Impact

1. **Architectural coupling**: Encourages developers to keep adding hardcoded tool handlers directly to the agent binary instead of writing separate, reusable, and decoupled MCP servers.
2. **Maintenance overhead**: Upgrading or adding new tools requires recompiling and redeploying the core agent service rather than simply registering a new MCP server CRD.

## Expected Behaviour

1. The `CognitiveEngine` MUST NOT contain or support hardcoded local tool function handlers (with the possible exception of infrastructural loop control/diagnostics tools if strictly necessary, such as LTM `recall_memory`).
2. All business-specific and computational tools (e.g. math operations, system integrations, APIs) MUST be delegated exclusively to external Model Context Protocol (MCP) servers.
3. The cognitive engine should discover, load, and execute tools dynamically using standard MCP protocol clients connected to resolved MCP servers.

## Acceptance Criteria

1. **Decoupled Architecture**:
   - Business/functional mock tools like `math_add` are removed from the agent's core binary and bootstrap files.
   - The cognitive engine only executes interactive actions by delegating to connected MCP servers over the standard Model Context Protocol.
2. **Mocks Removal**:
   - No programmatic tool handlers remain compiled in `cmd/agent/main.go`.
