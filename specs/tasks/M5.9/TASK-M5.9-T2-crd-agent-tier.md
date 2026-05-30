# TASK-M5.9-T2: Update Kubernetes CRD Agent Spec with Tier

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.9-T2                                |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M5.9                                |
| Depends On    | TASK-M5.9-T1                                |

## Description

Adds the `tier` field to the Agent CRD schema and updates the Keeper's CRD Coordinator to submit the tier to Kubernetes during deployment.

## Work Items

1. **RED Phase**:
   - Write integration tests in [coordinator_test.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/outbound/crd/coordinator_test.go) asserting that when submitting an Agent with tier `"heavy"`, the generated Custom Resource `Agent` spec has `spec.tier` set to `"heavy"`.
   - Verify that the tests fail when run against the existing codebase.
2. **GREEN Phase**:
   - Modify [agent_types.go](file:///Users/R.Pasquini/Projects/side/tacito-square/pkg/kubernetes/apis/tacito/v1alpha1/agent_types.go) to add `Tier string `json:"tier,omitempty"`` to the `AgentSpec` struct.
   - Run Operator code-generators (`make generate`) to update CRD manifests if required.
   - Update the mapping in [coordinator.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/outbound/crd/coordinator.go) to copy the agent's `Tier` to `AgentSpec.Tier` during custom resource construction.
   - Verify all tests compile and pass successfully.
3. **REFACTOR Phase**:
   - Clean up imports, comments, and struct mappings.

## Acceptance Criteria

1. Submitting an Agent CRD writes `spec.tier` correctly in Kubernetes Custom Resource objects.
