# TASK-M5.5.1: CRD Schema and Keeper Coordinator Integration

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.5.1                                 |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M5.5                                |
| Depends On    | none                                        |

## Description

Extend the database and API models to support `AllowedTools` whitelisting when binding MCP client configurations to agents. Extend the `TacitoAgent` Custom Resource Definition (CRD) schema with the `MCPClients` array spec. Implement the `crd_coordinator.go` logic inside the keeper to resolve full `MCPServer` records from Postgres, merge agent-specific overrides (`CustomEnv`, `CustomArgs`, `AllowedTools`), and submit the compiled specifications to the K8s custom resource.

## Work Items

1. **RED Phase**:
   * Write unit tests in `internal/keeper/adapters/outbound/crd/crd_coordinator_test.go` asserting that `SubmitAgentCRD` successfully loads associated MCP client configurations, merges command/args/env/url, and includes `AllowedTools` in the CRD payload.
   * Write tests in `internal/keeper/domain/model/agent_test.go` verifying that `MCPClientConfig` validates the `AllowedTools` parameters.
   * Verify test suite compilation and execution fails (RED).

2. **GREEN Phase**:
   * Add `AllowedTools []string` to the `MCPClientConfig` struct in `internal/keeper/domain/model/agent.go` and update its validation rules.
   * Update the GORM structure mapping and the `CreateMCPClient` schema in `internal/keeper/adapters/inbound/http/agent_handlers.go`.
   * Add `MCPClients []MCPClientSpec` to the `TacitoAgentSpec` struct in `pkg/kubernetes/apis/tacito/v1alpha1/tacitoagent_types.go`.
   * Implement the database-to-crd resolution logic inside `ResolveAndSynthesizeSystemPrompt` or `SubmitAgentCRD` in `internal/keeper/adapters/outbound/crd/crd_coordinator.go`.
   * Verify all keeper and model unit tests pass (GREEN).

3. **REFACTOR Phase**:
   * Ensure that the keeper domain layer has absolutely zero imports referencing the adapter layers, keeping database structures clean and pure.

## Acceptance Criteria

1. The `TacitoAgentSpec` schema incorporates the `MCPClients` field containing transport, command, args, environment, URL, and allowed tools.
2. The keeper CRD coordinator resolves PostgreSQL MCP server bindings and successfully submits the merged configurations to the K8s custom resource.
