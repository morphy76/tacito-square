# TASK-M4.1.3: Helm CRD Manifest & Registration

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M4.1.3                                 |
| Status        | PLANNED                                     |
| Spec          | SPEC-FR-M4.1                                |
| Depends On    | TASK-M4.1.1                                 |

## Description

Create and register the OpenAPI schema CustomResourceDefinition YAML manifest for the `TacitoAgent` resource. Place it within the standard Helm directory structure so that Kubernetes installs the custom schema automatically before deploying application resources.

## Boundary & Target Functions

- **Directory**: `tools/helm/tacito-square/crds`
- **File**: `tools/helm/tacito-square/crds/tacitoagents.yaml`

## Work Items

1. **RED Phase**:
   * Create a validation test script or dry-run command to verify that rendering the Helm chart locally does not error on custom resource definitions.

2. **GREEN Phase**:
   * Construct `tacitoagents.yaml` under `tools/helm/tacito-square/crds/` with complete `apiextensions.k8s.io/v1` syntax:
     * Expose group `tacito.square.io`, plural `tacitoagents`, singular `tacitoagent`, kind `TacitoAgent`.
     * Expose OpenAPI validation block mapping schema limits:
       * `tenantId` (string, required, minLength: 1).
       * `agentName` (string, required, minLength: 1).
       * `communityRef` (string, required, minLength: 1).
       * `llmConfig` (object, required, model standard properties).
       * `llmConfig.temperature` (number, minimum: 0.0, maximum: 2.0, default: 0.7).
       * `llmConfig.maxTokens` (integer, minimum: 1, maximum: 8192, default: 2048).
       * `replicas` (integer, minimum: 0, maximum: 10, default: 1).
     * Expose scale subresource config:
       ```yaml
       subresources:
         status: {}
         scale:
           specReplicasPath: .spec.replicas
           statusReplicasPath: .status.replicas
           labelSelectorPath: .status.selector
       ```

3. **REFACTOR Phase**:
   * Validate that the manifest has correct indentation and clean metadata comments.

## Acceptance Criteria

1. Running `helm template` outputs valid YAML and lists the registered CustomResourceDefinition schema manifest.
2. OpenAPI validation block matches spec boundary ranges.
