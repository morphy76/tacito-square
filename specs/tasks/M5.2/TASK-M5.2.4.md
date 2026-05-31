# TASK-M5.2.4: Dev Mock Servers & Helm Chart Configurations

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.2.4                                 |
| Status        | VERIFIED                                    |
| Spec          | SPEC-FR-M5.2                                |
| Depends On    | TASK-M5.2.3                                 |

## Description

Deploy open-source mock servers for OpenAI and Ollama APIs by default inside the standalone agent Helm chart `tools/helm/tacito-agent` to support offline development.

## Work Items

1. **RED Phase**:
   - Write a template validation assertion in `test/helm/test_agent_standalone_chart.sh` checking that rendering the Helm chart generates Deployment, Service, and ConfigMap definitions for OpenAI and Ollama mock backends.
   - Run the validation script and observe expected failures (RED).

2. **GREEN Phase**:
   - Update `tools/helm/tacito-agent/Chart.yaml` as required.
   - Create `tools/helm/tacito-agent/templates/mocks.yaml` containing the deployment and service definitions for standard open-source API mocking tools (e.g. standard MockServer or multi-endpoint mocking sidecars).
   - Configure default values in `tools/helm/tacito-agent/values.yaml` targeting these local mock services by default for the brain adapter's API endpoints.
   - Verify dry-run rendering successfully renders both the agent and mock server workloads (GREEN).

3. **REFACTOR Phase**:
   - Organize and document value options clearly inside `tools/helm/tacito-agent/values.yaml`.

## Acceptance Criteria

1. Running `helm template` renders valid Kubernetes mock server deployments and services by default.
2. The agent's target LLM endpoint values point to these local mock service names by default.
