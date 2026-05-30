# TASK-M5.7.2: Deployment Template and Environment Injection

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.7.2                                 |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M5.7                                |
| Depends On    | TASK-M5.7.1                                 |

## Description

Implement the core `templates/deployment.yaml` template in the standalone agent Helm chart. The deployment MUST run the standard agent container image, inject configuration environment variables from the generated ConfigMap via `envFrom`, reference LLM credentials securely from a Kubernetes Secret, and establish standard Liveness (`/healthz`) and Readiness (`/readyz`) probes.

## Work Items

1. **RED Phase**:
   - Write test assertions in `test/helm/test_agent_standalone_chart.sh` expecting a Deployment resource named `my-agent-tacito-agent` with:
     - An `envFrom` block targeting the generated ConfigMap.
     - Liveness and Readiness HTTP probes pointing to `/healthz` and `/readyz` at the configured port.
     - An `env` definition referencing the secret named in `agent.llm.credentialsSecret` under `TS_AGENT_LLM_API_KEY` (or the appropriate credentials key).
   - Run the script and observe failure (RED) because the deployment template does not exist.

2. **GREEN Phase**:
   - Create `tools/helm/tacito-agent/templates/deployment.yaml`.
   - Configure a standard deployment template utilizing helper labels and names.
   - Inject the ConfigMap using `envFrom`.
   - Setup secure credentials mapping: if `agent.llm.credentialsSecret` is provided, mount it as `TS_AGENT_LLM_API_KEY` (or equivalent) in the environment.
   - Configure the standard livenessProbe and readinessProbe targeting the HTTP port, with customizable `initialDelaySeconds`, `periodSeconds`, and `timeoutSeconds`.
   - Wire standard resource requirements, `nodeSelector`, `tolerations`, and `affinity` blocks.
   - Run tests and verify dry-run template rendering outputs a valid, conformant Deployment (GREEN).

3. **REFACTOR Phase**:
   - Clean up templating syntax, formatting, and indentation.
   - Ensure liveness and readiness probe parameters are fully customizable via `values.yaml`.

## Acceptance Criteria

1. Running `helm template` successfully renders a Deployment resource with the correct label structure and container image reference.
2. The Deployment includes an `envFrom` referencing the generated ConfigMap.
3. Standard Liveness and Readiness probes are configured with default ports and customizable bounds.
4. If specified, LLM credentials are bound to a secret reference instead of plain values.
