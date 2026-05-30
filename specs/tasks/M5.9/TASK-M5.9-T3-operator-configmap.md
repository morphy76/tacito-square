# TASK-M5.9-T3: Helm Values and ConfigMap Setup for Mapped Tiers

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.9-T3                                |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M5.9                                |
| Depends On    | TASK-M5.9-T2                                |

## Description

Creates the Helm configurations in `values.yaml` and a template to dynamically generate the `ts-agent-tiers` ConfigMap and mount it to the Operator during deployment.

## Work Items

1. **RED Phase**:
   - Write a quick shell/go dry-run verification target to confirm ConfigMap volume mounts.
2. **GREEN Phase**:
   - Add default fallback tier `agentTiers.default: "standard"` and configurations `agentTiers.configs` to the parent [values.yaml](file:///Users/R.Pasquini/Projects/side/tacito-square/deploy/helm/tacito-square/values.yaml) file.
   - Create a new Helm template `deploy/helm/tacito-square/templates/operator/configmap-tiers.yaml` that serializes the configuration as a YAML/JSON block in a ConfigMap named `ts-agent-tiers`.
   - Modify the Operator deployment template `deploy/helm/tacito-square/templates/operator/deployment.yaml` to mount this ConfigMap at `/etc/tacito/tiers/` inside the Operator container.
   - Run `make helm-template` to verify formatting and output success.
3. **REFACTOR Phase**:
   - Consolidate and clean up the Helm values indentation.

## Acceptance Criteria

1. Running `helm template` outputs a valid `ts-agent-tiers` ConfigMap and correctly mounts it as a volume to the Operator pod.
