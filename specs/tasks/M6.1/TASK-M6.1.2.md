# TASK-M6.1.2: Operator CRD Spec & Deployment Configuration

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M6.1.2                                 |
| Status        | VERIFIED                                    |
| Spec          | SPEC-FR-M6.1                                |
| Depends On    | none                                        |

## Description

Add the topological `role` field to the `TacitoAgentSpec` custom resource definition. Update the Operator Reconciler to read this field and inject it into the agent container's environment variables as `TS_AGENT_ROLE`.

## Boundary & Target Functions

- **Package**: `pkg/kubernetes/apis/tacito/v1alpha1`, `internal/operator/application/service`
- **Files**:
  - `pkg/kubernetes/apis/tacito/v1alpha1/tacitoagent_types.go`
  - `internal/operator/application/service/reconcile_service.go`

## Work Items

1. **RED Phase (Write Tests First)**:
   * Write unit tests in `internal/operator/application/service/reconcile_service_test.go`:
     * Assert that `BuildDeployment` maps `spec.role = "hub"` to the environment variable `TS_AGENT_ROLE=hub`.
     * Assert that `BuildDeployment` defaults to `TS_AGENT_ROLE=spoke` if the role field is empty or omitted.
     * Verify that CRD validation reject values other than `hub` and `spoke` for `spec.role`.

2. **GREEN Phase (Implement Minimum Code)**:
   * Update the `TacitoAgentSpec` struct in `tacitoagent_types.go` to include `Role string json:"role,omitempty"`. Add validation kubebuilder annotations.
   * Run code generation scripts/make targets (`make generate` or `make manifests`) to rebuild CRD YAML schemas.
   * Modify `BuildDeployment` in `reconcile_service.go` to extract `agent.Spec.Role` and inject `TS_AGENT_ROLE` into the pod's template container environment variables.

3. **REFACTOR Phase (Clean & Generalize)**:
   * Clean up container spec generation and ensure it satisfies the hexagonal architecture.

## Acceptance Criteria

1. Reconciler unit tests assert correct environment variable propagation.
2. Regenerated CRD manifests (`tools/helm/tacito-square/crds/`) include the validation rules for the `role` property.
