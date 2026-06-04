# TASK-M5.9-T3: Helm Values and ConfigMap Setup for Mapped Tiers

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.9-T3                                |
| Status        | IMPLEMENTED                                 |
| Spec          | SPEC-FR-M5.9                                |
| Depends On    | TASK-M5.9-T2                                |

## Description

Creates the Helm configurations in `values.yaml` and a template to dynamically generate the `ts-agent-tiers` ConfigMap and mount it into the Operator pod at startup.

## Work Items

1. **RED Phase**:
   - Write a `helm template` dry-run verification and confirm the ConfigMap is rendered with the expected tier entries and that the Operator deployment includes the correct volume and volumeMount.
2. **GREEN Phase**:
   - Add an `agentTiers.configs` map to [values.yaml](file:///Users/R.Pasquini/Projects/side/tacito-square/tools/helm/tacito-square/values.yaml) containing at least one example named tier (e.g. `heavy`) with its image, resources, pullPolicy, and probe overrides. The implicit default (from `agent.*` values) requires no extra key.
   - Create a new Helm template [configmap-tiers.yaml](file:///Users/R.Pasquini/Projects/side/tacito-square/tools/helm/tacito-square/templates/agent/configmap-tiers.yaml) that serializes `agentTiers.configs` into a `ConfigMap` named `ts-agent-tiers`.
   - Modify the Operator deployment template [operator-deployment.yaml](file:///Users/R.Pasquini/Projects/side/tacito-square/tools/helm/tacito-square/templates/agent/operator-deployment.yaml) to mount `ts-agent-tiers` as a volume at `/etc/tacito/tiers/tiers.yaml` inside the Operator container.
   - Run `make helm-template` to verify formatting and rendered output.
3. **REFACTOR Phase**:
   - Consolidate and clean up the Helm values indentation and comments.

## Acceptance Criteria

1. Running `helm template` outputs a valid `ts-agent-tiers` ConfigMap and correctly mounts it as a volume at `/etc/tacito/tiers/tiers.yaml` inside the Operator pod.
2. At least one example named tier is present in the default `values.yaml`.
