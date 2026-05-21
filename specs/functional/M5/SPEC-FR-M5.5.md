# SPEC-FR-M5.5: Tool Invocation (MCP)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M5.5                                |
| Status        | DRAFT                                       |
| Milestone     | M5                                          |
| Component     | agent                                       |
| Depends On    | SPEC-FR-M5.2                                |
| Supersedes    | none                                        |

## Context

Agents invoke external tools via MCP (Model Context Protocol) during reasoning loops. Tools extend agent capabilities beyond pure LLM reasoning — e.g., web search, database queries, API calls.

## Specification

1. The system MUST define a `ToolExecutor` outbound port in the agent domain layer.
2. The system MUST implement an MCP client adapter using the MCP Go SDK (per SPEC-NFR-STACK).
3. The adapter MUST support tool discovery (list available tools from MCP server).
4. The adapter MUST support tool execution with structured input/output.
5. Tool results MUST be integrated back into the LLM reasoning loop.
6. Tool invocation MUST be traced via OpenTelemetry (per SPEC-NFR-REACTIVE).
7. Circuit breakers MUST protect against unresponsive MCP servers (per SPEC-NFR-CLOUD).

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
