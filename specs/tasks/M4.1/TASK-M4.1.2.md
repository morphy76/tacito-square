# TASK-M4.1.2: DeepCopy Implementation for TacitoAgent Scheme

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M4.1.2                                 |
| Status        | PLANNED                                     |
| Spec          | SPEC-FR-M4.1                                |
| Depends On    | TASK-M4.1.1                                 |

## Description

Implement standard `DeepCopy` and `DeepCopyInto` Go methods for all custom types under the shared package API to ensure `runtime.Object` compatibility. This guarantees that controller-runtime managers and Kubernetes client caches can clone custom resources safely without panic, compiling seamlessly even when external code generation tools are absent.

## Boundary & Target Functions

- **Package**: `pkg/kubernetes/apis/tacito/v1alpha1`
- **File**: `pkg/kubernetes/apis/tacito/v1alpha1/zz_generated.deepcopy.go`
- **Target Functions**:
  - `(*TacitoAgent) DeepCopy() *TacitoAgent`
  - `(*TacitoAgent) DeepCopyInto(out *TacitoAgent)`
  - `(*TacitoAgentSpec) DeepCopy() *TacitoAgentSpec`
  - `(*TacitoAgentSpec) DeepCopyInto(out *TacitoAgentSpec)`
  - `(*TacitoAgentStatus) DeepCopy() *TacitoAgentStatus`
  - `(*TacitoAgentStatus) DeepCopyInto(out *TacitoAgentStatus)`

## Work Items

1. **RED Phase**:
   * Create test cases in `pkg/kubernetes/apis/tacito/v1alpha1/deepcopy_test.go` to assert:
     * Deep copying a non-nil `TacitoAgent` struct yields a fully isolated duplicate with equal value but distinct pointers.
     * Modifying field properties on a cloned instance does not alter the original instance state.

2. **GREEN Phase**:
   * Write custom manual `DeepCopy` and `DeepCopyInto` functions inside `zz_generated.deepcopy.go` implementing recursive copying of nested structs (including brain configs, CPU/Memory resource allocations, slices, and conditions).
   * Ensure that the deepcopy helper functions carry appropriate package definitions and register `DeepCopyObject` cleanly.

3. **REFACTOR Phase**:
   * Remove any redundant allocations or reflections from the deepcopy loop to maximize performance during controller-runtime marshals.

## Acceptance Criteria

1. DeepCopy test suite passes successfully.
2. Go structs implement `runtime.Object` and compile cleanly with `sigs.k8s.io/controller-runtime` and `k8s.io/apimachinery` manager schemas.
