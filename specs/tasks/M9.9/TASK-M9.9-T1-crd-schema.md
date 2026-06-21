# TASK-M9.9-T1: CRD Schema Extension

| Field       | Value                                      |
|-------------|--------------------------------------------|
| Task ID     | TASK-M9.9-T1                               |
| Spec        | SPEC-FR-M9.9                               |
| Boundary    | CRD Type System                            |
| Status      | TODO                                       |
| Depends On  | —                                          |

## Objective

Extend `TacitoAgentSpec` with the four new fields required for zero-scaling configuration and regenerate the deepcopy code.

## Files

| File | Action |
|------|--------|
| `pkg/kubernetes/apis/tacito/v1alpha1/tacitoagent_types.go` | MODIFY |
| `pkg/kubernetes/apis/tacito/v1alpha1/zz_generated.deepcopy.go` | MODIFY (regenerated) |

## RED Phase

Add unit tests to `pkg/kubernetes/apis/tacito/v1alpha1/tacitoagent_types_test.go` asserting:
- A `TacitoAgentSpec` with no zero-scaling fields marshals/unmarshals with correct kubebuilder defaults (`scaleToZeroEnabled=false`, `minReplicas=1`, `maxReplicas=1`, `idleTimeoutSeconds=300`).
- A `TacitoAgentSpec` with `minReplicas=2, maxReplicas=5, idleTimeoutSeconds=60, scaleToZeroEnabled=true` round-trips correctly through JSON.
- DeepCopy of a spec with all four new fields populated produces an independent copy (no shared pointer references).

Run `make test` — tests must fail (RED).

## GREEN Phase

Add the following fields to `TacitoAgentSpec` in `tacitoagent_types.go`:

```go
// ScaleToZeroEnabled enables automatic scale-to-zero when the agent is idle.
// +optional
// +kubebuilder:default=false
ScaleToZeroEnabled *bool `json:"scaleToZeroEnabled,omitempty"`

// MinReplicas is the minimum number of replicas. Used as the scale-up target after idle scale-down.
// +optional
// +kubebuilder:validation:Minimum=0
// +kubebuilder:validation:Maximum=10
// +kubebuilder:default=1
MinReplicas *int32 `json:"minReplicas,omitempty"`

// MaxReplicas is the maximum number of replicas.
// +optional
// +kubebuilder:validation:Minimum=1
// +kubebuilder:validation:Maximum=10
// +kubebuilder:default=1
MaxReplicas *int32 `json:"maxReplicas,omitempty"`

// IdleTimeoutSeconds is the duration of inactivity (in seconds) before the agent is scaled to zero.
// Requires ScaleToZeroEnabled=true to take effect. Minimum 30 seconds.
// +optional
// +kubebuilder:validation:Minimum=30
// +kubebuilder:default=300
IdleTimeoutSeconds *int32 `json:"idleTimeoutSeconds,omitempty"`
```

Regenerate deepcopy:
```bash
make generate
```

Run `make test` — tests must pass (GREEN).

## REFACTOR Phase

- Verify kubebuilder markers are correct by running `make generate` a second time and confirming no diff.
- Ensure all existing tests in `tacitoagent_types_test.go` and `deepcopy_test.go` remain GREEN.
