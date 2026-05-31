# TASK-M5.7.4: Add Standalone Agent Client CLI Component

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.7.4                                 |
| Status        | VERIFIED                                    |
| Spec          | SPEC-FR-M5.7                                |
| Depends On    | TASK-M5.7.3                                 |

## Description

Add a CLI helper component to the standalone agent Helm chart that allows the user to easily send messages to the agent and receive responses using the NATS CLI client. The component will be deployed as an executable wrapper script (`tacito-client`) mounted from a Kubernetes ConfigMap into the client container.

## Work Items

1. **RED Phase**:
   - Write test assertions in `test/helm/test_agent_standalone_chart.sh` expecting:
     - A new ConfigMap resource named `my-agent-tacito-agent-client-config` containing the `tacito-client` executable script.
     - The client Deployment mounting the ConfigMap volume to `/usr/local/bin/` with executable permissions (`0755`).
   - Run the script and observe failure (RED).

2. **GREEN Phase**:
   - Create `tools/helm/tacito-agent/templates/client-configmap.yaml` wrapping a shell script that parses inputs and invokes `nats request` on the agent's subject.
   - Modify `tools/helm/tacito-agent/templates/client-deployment.yaml` to mount the script to `/usr/local/bin/tacito-client` with `defaultMode: 0755`.
   - Verify all tests pass successfully (GREEN).

3. **REFACTOR Phase**:
   - Refactor the shell wrapper script to include usage documentation, error checks for NATS connections, and clean logging outputs.

## Acceptance Criteria

1. Dry-run template rendering outputs the client ConfigMap and correct volume mounting.
2. The `tacito-client` script is fully configured to execute `nats request` to the target agent.
