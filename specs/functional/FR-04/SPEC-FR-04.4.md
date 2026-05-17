# SPEC-FR-04.4: Tool Invocation (MCP Adapter)

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-FR-04.4                       |
| Status        | IMPLEMENTED                        |
| Milestone     | M2                                 |
| FR/NFR Ref    | FR-04.4                            |
| Component     | agent                              |
| Depends On    | SPEC-FR-04.1                       |

## Context

Agents extend their capabilities by invoking external tools via MCP (Model Context Protocol). The adapter translates between the domain Tools port and MCP server interactions.

## Specification

1. `Tools` outbound port (in `ports.go`) defines:
   - `Invoke(ctx, toolName, input)` — calls a tool and returns string output
   - `ListTools(ctx)` — returns available `ToolDescriptor` list
2. `MCPClient` interface abstracts MCP server operations:
   - `CallTool(ctx, toolName, input)` — invokes a tool on the MCP server
   - `ListTools(ctx)` — returns `[]MCPToolInfo` from the server
3. `ToolsAdapter` MUST:
   a. Delegate `Invoke` to `MCPClient.CallTool`
   b. Propagate tool-not-found and server errors with context
   c. Map `MCPToolInfo` to `domain.ToolDescriptor` (name + description)
4. `MCPToolInfo` carries `Name`, `Description`, `InputSchema`.

## Acceptance Criteria

1. Invoke succeeds and returns tool output ✅
2. Invoke with unknown tool returns error containing "tool not found" ✅
3. Invoke with server error propagates error ✅
4. ListTools returns correct count of ToolDescriptors ✅
5. ListTools includes name and description for each tool ✅
6. Returned descriptors match domain.ToolDescriptor type ✅

## Files

- `internal/agent/ports/outbound/ports.go` ✅ (Tools interface)
- `internal/agent/adapters/outbound/mcp/tools_adapter.go` ✅ IMPLEMENTED
- `internal/agent/adapters/outbound/mcp/tools_adapter_test.go` ✅ 6 tests
