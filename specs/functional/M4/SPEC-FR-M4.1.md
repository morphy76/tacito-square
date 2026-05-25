# SPEC-FR-M4.1: Agent CRD Definition & Registration

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M4.1                                |
| Status        | ACCEPTED                                    |
| Milestone     | M4                                          |
| Component     | operator                                    |
| Depends On    | none                                        |
| Supersedes    | none                                        |

## Context

The operator needs a CustomResourceDefinition (CRD) for `TacitoAgent` resources in the `tacito.square.io/v1alpha1` API group. This is the foundational resource that represents an agent instance in the cluster, derived from a persisted agent model.

## Specification

1. **CRD Definition**:
   * The system MUST define a `TacitoAgent` CRD inside the API group `tacito.square.io/v1alpha1`.
   * The CRD MUST be namespace-scoped.

2. **CRD Spec Fields & Validations (Kubebuilder Markers)**:
   * `tenantId` (string, required): The ID of the tenant owning this agent (Option B for multi-tenancy). Minimum length: 1.
   * `agentName` (string, required): The unique name of the agent. Minimum length: 1.
   * `communityRef` (string, required): Reference to the parent `TacitoCommunity` ID/name. Minimum length: 1.
   * `llmConfig` (object, required): LLM parameters for the agent's brain:
     * `model` (string, required): The model identifier (e.g., `gpt-4o`, `gemini-1.5-pro`).
     * `temperature` (number, optional): Minimum: `0.0`, Maximum: `2.0`, Default: `0.7`.
     * `maxTokens` (integer, optional): Minimum: `1`, Maximum: `8192`, Default: `2048`.
   * `systemPrompt` (string, optional): The fully synthesized/composed system prompt (compiled from agent description, prompt collections, and skill collections).
   * `resources` (object, optional): Kubernetes core CPU/Memory request and limit constraints.
   * `replicas` (integer, optional): Desired count of active pods. Minimum: `0`, Maximum: `10`, Default: `1`.

3. **CRD Status**:
   * `phase` (string): Standard phase tracking. Enum: `Pending`, `Running`, `Idle`, `Terminated`.
   * `conditions` (array of metav1.Condition): Standard Kubernetes conditions (e.g., `Available`, `Reconciling`, `Failed`).
   * `lastHeartbeat` (metav1.Time): High-resolution timestamp of the last agent heartbeat.

4. **Autoscaling Hooks**:
   * The CRD spec and status MUST be configured to support the standard Kubernetes `/scale` subresource (Option A):
     ```go
     // +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.selector
     ```
     *(Note: Full HPA metrics autoscaling reconciliation loops are postponed, but the CRD structure must support it natively).*

5. **CRD Generation & Helm Registration**:
   * CRD Go structs and validation markers MUST be defined under `pkg/kubernetes/apis/tacito/v1alpha1` (shared with the Keeper component).
   * The manifest MUST be generated using `controller-gen` (part of Kubebuilder tools).
   * The generated CRD YAML file MUST be stored and installed via the Helm application chart under `tools/helm/tacito-square/templates/agent/crds/` or using the standard Helm `crds/` directory.

## Acceptance Criteria

1. **CRD Schema Compliance**:
   * Deploying a `TacitoAgent` resource without a `tenantId` or with an invalid `temperature` (e.g., `3.0`) MUST be rejected by the Kubernetes API server validation webhook.
2. **Multi-Tenancy Propagation**:
   * The operator's Go structures MUST accurately bind `spec.tenantId` and propagate it during serialization.
3. **Scale Subresource Verification**:
   * Running `kubectl scale tacitoagent <name> --replicas=3` MUST succeed and update the resource's `spec.replicas`.
4. **Operator Probes & Readiness**:
   * Since the Operator runs as a Deployment and exposes an HTTP listener (port `8082`), it MUST implement standard `/healthz` and `/readyz` probes.
   * The `/readyz` probe MUST perform an asynchronous, parallel check on the Kubernetes API Server client connectivity. If the Kube-API is unreachable, it MUST return `503 Service Unavailable`.

## Test Plan

### Automated Tests
* **Envtest Integration Tests**:
  * Implement tests using `sigs.k8s.io/controller-runtime/pkg/envtest`.
  * Spin up a local mock API server, register the generated `TacitoAgent` CRD, and verify that creating valid/invalid resources behaves exactly as specified.
  * Verify `/readyz` behavior when the Kube client is mocked to be connected versus disconnected.
* **Makefile Integration**:
  * Execute via `make test-operator`.

### Manual Verification
* Deploy the CRD to a local Minikube/Kind cluster and run schema validation tests by submitting sample YAML manifests.

## Files Affected

* `go.mod` (Register new dependencies: `sigs.k8s.io/controller-runtime` and `k8s.io/apimachinery`)
* `pkg/kubernetes/apis/tacito/v1alpha1/groupversion_info.go` [NEW]
* `pkg/kubernetes/apis/tacito/v1alpha1/tacitoagent_types.go` [NEW]
* `pkg/kubernetes/apis/tacito/v1alpha1/zz_generated.deepcopy.go` [NEW]
* `tools/helm/tacito-square/crds/tacitoagents.yaml` [NEW]
* `internal/operator/bootstrap.go` (Enhance `/readyz` probe to check Kube client)

