# TASK-M5.5.2: Operator Reconciler and Developer Helm Chart Serialization

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.5.2                                 |
| Status        | IMPLEMENTED                                 |
| Spec          | SPEC-FR-M5.5                                |
| Depends On    | TASK-M5.5.1                                 |

## Description

Implement the operator side of the MCP client configuration propagation. Update the agent Deployment builder in the operator's reconcile service to serialize the `MCPClients` custom resource spec as a structured JSON string and inject it as `TS_AGENT_MCP_CLIENTS`. Update the standalone developer Helm chart to serialize `mcpClients` configurations from `values.yaml` into the same environment variable layout, ensuring parity between local development and production.

## Work Items

1. **RED Phase**:
   * Write operator unit tests in `internal/operator/application/service/reconcile_service_test.go` asserting that `BuildDeployment` maps `Spec.MCPClients` to the `TS_AGENT_MCP_CLIENTS` JSON environment variable.
   * Write dry-run Helm rendering checks in `test/helm/test_agent_standalone_chart.sh` asserting that supplying `mcpClients` in `values.yaml` generates a deployment manifest containing the serialized `TS_AGENT_MCP_CLIENTS` JSON string in the pod's environment.
   * Run the tests and script to verify expected failures (RED).

2. **GREEN Phase**:
   * Modify the operator's deployment builder in `internal/operator/application/service/reconcile_service.go` to serialize `agent.Spec.MCPClients` into JSON and append it to the container `env` slice.
   * Update the developer's standalone Helm chart `tools/helm/tacito-agent/values.yaml` to declare default structure blocks for developer-level MCP clients.
   * Update the deployment template `tools/helm/tacito-agent/templates/deployment.yaml` to serialize `values.mcpClients` to JSON using Helm template functions (e.g. `toJson`) and output it as the `TS_AGENT_MCP_CLIENTS` environment variable.
   * Verify all tests and Helm dry-runs pass successfully (GREEN).

3. **REFACTOR Phase**:
   * Review Helm indentation and JSON escaping in `deployment.yaml` to prevent invalid YAML generation when `AllowedTools` or overrides contain special characters.

## Acceptance Criteria

1. The K8s operator injects `TS_AGENT_MCP_CLIENTS` as a single, valid JSON-serialized environment variable into the agent Pod deployment.
2. The standalone developer Helm chart produces the exact same environment layout when `mcpClients` is populated in its values.
