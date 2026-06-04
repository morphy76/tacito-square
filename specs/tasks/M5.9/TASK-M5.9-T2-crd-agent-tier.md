# TASK-M5.9-T2: Update Kubernetes CRD Agent Spec with Tier

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.9-T2                                |
| Status        | VERIFIED                                    |
| Spec          | SPEC-FR-M5.9                                |
| Depends On    | TASK-M5.9-T1                                |

## Description

Adds the `tier` field to the `TacitoAgentSpec` CRD struct, removes the raw `Resources` field (now determined by tier), and updates the Keeper's CRD Coordinator to propagate the tier value into the Kubernetes CR at submission time.

## Work Items

1. **RED Phase**:
   - Write integration tests in [crd_coordinator_test.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/outbound/crd/crd_coordinator_test.go) asserting that submitting an Agent with `Tier = "heavy"` generates a CR with `spec.tier = "heavy"`, and that submitting an agent with empty tier generates a CR with an empty `spec.tier`.
   - Verify that the tests fail against the existing codebase.
2. **GREEN Phase**:
   - In [agent_types.go](file:///Users/R.Pasquini/Projects/side/tacito-square/pkg/kubernetes/apis/tacito/v1alpha1/agent_types.go): add `Tier string` (`// +optional`, `json:"tier,omitempty"`) and **remove** `Resources *corev1.ResourceRequirements` from `TacitoAgentSpec` (resources are now entirely controlled by the Operator tier map).
   - Run `make generate` to regenerate [zz_generated.deepcopy.go](file:///Users/R.Pasquini/Projects/side/tacito-square/pkg/kubernetes/apis/tacito/v1alpha1/zz_generated.deepcopy.go).
   - Update [crd_coordinator.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/outbound/crd/crd_coordinator.go): remove the `Resources` field mapping in `SubmitAgentCRD` and populate `Spec.Tier = agent.Tier` on both create and update paths.
   - Verify all tests compile and pass.
3. **REFACTOR Phase**:
   - Remove any remaining references to `spec.Resources` in CRD tests or fixtures.

## Acceptance Criteria

1. Submitting an Agent CRD writes `spec.tier` correctly in Kubernetes CRs.
2. `TacitoAgentSpec` no longer exposes a `Resources` field — resources are exclusively set by the Operator at reconciliation time.
