# TASK-M4.1.1: Shared Kubernetes API Group & Custom Resource Scheme

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M4.1.1                                 |
| Status        | PLANNED                                     |
| Spec          | SPEC-FR-M4.1                                |
| Depends On    | none                                        |

## Description

Define the shared Custom Resource types and scheme for `TacitoAgent` under the API group `tacito.square.io/v1alpha1`. Struct definitions must carry complete Kubebuilder markers to enforce schema validation ranges and default values, ensuring standard controller-runtime compatibility without circular imports between the keeper and operator.

## Boundary & Target Functions

- **Package**: `pkg/kubernetes/apis/tacito/v1alpha1`
- **Files**:
  - `pkg/kubernetes/apis/tacito/v1alpha1/groupversion_info.go`
  - `pkg/kubernetes/apis/tacito/v1alpha1/tacitoagent_types.go`
- **Target Structs**:
  - `TacitoAgent` (type root)
  - `TacitoAgentSpec` (spec configuration)
  - `TacitoAgentStatus` (status phase & conditions)

## Work Items

1. **RED Phase**:
   * Write validation test cases in `pkg/kubernetes/apis/tacito/v1alpha1/tacitoagent_types_test.go` to assert:
     * Spec fields bind correctly from JSON payloads.
     * Required field tags represent schema constraints (`json:"tenantId"`, `json:"agentName"`, etc.).
     * Subresource scale markers are correctly annotated to map `.spec.replicas` and `.status.replicas`.

2. **GREEN Phase**:
   * Create `groupversion_info.go` defining the `SchemeGroupVersion` for API group `tacito.square.io` and version `v1alpha1`, registering types to the `SchemeBuilder`.
   * Create `tacitoagent_types.go` declaring the `TacitoAgentSpec` struct:
     * `tenantId` (string, required)
     * `agentName` (string, required)
     * `communityRef` (string, required)
     * `llmConfig` (Brain config struct: model, temperature, maxTokens)
     * `systemPrompt` (string, optional)
     * `replicas` (int, optional)
     * `resources` (optional core requests/limits)
   * Declare the `TacitoAgentStatus` struct holding `phase`, `conditions` (`[]metav1.Condition`), and `lastHeartbeat` (`metav1.Time`).
   * Apply appropriate `// +kubebuilder:object:root=true`, `// +kubebuilder:subresource:status`, and `// +kubebuilder:subresource:scale` comment markers.

3. **REFACTOR Phase**:
   * Clean up Go struct field alignments to optimize memory allocation.
   * Verify all tags adhere to camelCase standards matching standard K8s API designs.

## Acceptance Criteria

1. Shared Custom Resource structs compile cleanly with all standard meta/v1 package dependencies.
2. Go structs contain required json tags matching the spec.
3. Scale and status subresources are properly decorated with Kubebuilder markers.
