# TASK-M6.5.1.5: Operator CRD — TacitoAgentSpec Role Field Update

| Field       | Value |
|-------------|-------|
| ID          | TASK-M6.5.1.5 |
| Status      | IMPLEMENTED |
| Spec        | SPEC-FR-M6.5.1 |
| Depends On  | TASK-M6.5.1.1 |

## Description

Update the `TacitoAgent` CRD spec in `pkg/kubernetes/apis/tacito/v1alpha1/tacitoagent_types.go` to reflect the extended role enum (`hub`, `spoke`, `standalone`) and remove the static `+kubebuilder:default="spoke"` marker, since role is now sourced from the `CommunityAssignment` record at Keeper reconciliation time rather than being a static agent property.

## Work Items

1. **RED Phase**:
   - Update `pkg/kubernetes/apis/tacito/v1alpha1/tacitoagent_types_test.go`:
     - Add test asserting that a `TacitoAgentSpec` with `Role = "standalone"` round-trips correctly through JSON marshal/unmarshal.
     - Verify that an empty `Role` field (omitempty) serializes without the key.
     - Keep existing `hub` and `spoke` round-trip tests.

2. **GREEN Phase**:
   - Modify `pkg/kubernetes/apis/tacito/v1alpha1/tacitoagent_types.go`:
     - Replace the kubebuilder markers on the `Role` field:
       ```go
       // Role is the topology role of the agent as assigned by Keeper ("hub", "spoke", or "standalone").
       // Populated at reconciliation time from the community_assignments record; do not set manually.
       // +optional
       // +kubebuilder:validation:Enum=hub;spoke;standalone
       Role string `json:"role,omitempty"`
       ```
     - Remove the `+kubebuilder:default="spoke"` marker.

3. **REFACTOR Phase**:
   - Update the godoc comment to make it explicit that this field is reconciler-managed and must not be set by human operators directly.
   - Run `make generate` to regenerate CRD manifests and confirm the updated enum is reflected in the generated YAML.

## Acceptance Criteria

1. `tacitoagent_types_test.go` passes GREEN with `standalone` round-trip test added.
2. The `+kubebuilder:default="spoke"` marker is fully removed from `Role`.
3. The `+kubebuilder:validation:Enum` marker lists exactly `hub;spoke;standalone`.
4. `make generate` completes without errors and the generated CRD YAML reflects the updated enum.
