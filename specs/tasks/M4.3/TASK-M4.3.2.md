# TASK-M4.3.2: Reconciler Pod Configuration & Multi-Tenancy Injection

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M4.3.2                                 |
| Status        | IMPLEMENTED                                 |
| Spec          | SPEC-FR-M4.3                                |
| Depends On    | TASK-M4.3.1                                 |

## Description

Implement the Deployment generation logic that merges base template configurations with custom spec attributes. The generator must inject the multi-tenancy context `spec.tenantId` directly into the pod container environment variable `TENANT_ID`, register base connection parameters (NATS, Redis, Qdrant, OTEL), map container CPU/Memory limits, and attach OwnerReferences to enforce automatic cascading resource garbage collection.

## Boundary & Target Functions

- **Package**: `internal/operator/application/service`
- **File**: `internal/operator/application/service/reconcile_service.go`
- **Target Functions**:
  - `(s *ReconcileAgentService) BuildDeployment(ctx context.Context, agent *v1alpha1.TacitoAgent) (*appsv1.Deployment, error)`
  - `(s *ReconcileAgentService) BuildHeadlessService(ctx context.Context, agent *v1alpha1.TacitoAgent) (*corev1.Service, error)`

## Work Items

1. **RED Phase**:
   * Implement unit tests inside `internal/operator/application/service/reconcile_service_test.go` to assert:
     * `BuildDeployment` maps `spec.tenantId` correctly to environment variable `TENANT_ID`.
     * `BuildDeployment` maps `spec.llmConfig` variables (`model`, `temperature`, `maxTokens`) and `spec.systemPrompt` accurately.
     * Container resource limits and requests match the parameters defined in `spec.resources`.
     * The generated Deployment includes `OwnerReferences` targeting the parent custom resource correctly.
     * `BuildHeadlessService` generates a valid Service configuration targeting port `8081` with ClusterIP set to `None`.

2. **GREEN Phase**:
   * Write `BuildDeployment` implementation:
     * Load global connections strings (NATS, Redis, Qdrant, OTEL endpoints) via Viper configurations.
     * Construct Pod containers mapping all configurations as environment variables (e.g. `TS_AGENT_NAME`, `TS_AGENT_BRAIN_MODEL`, etc.).
     * Bind resource boundaries directly to container requests/limits config blocks.
   * Write `BuildHeadlessService` constructing the cluster Service config.

3. **REFACTOR Phase**:
   * Optimize JSON serialization and environment maps.
   * Ensure that the distroless container image metadata references standard project tags (`0.1.0`).

## Acceptance Criteria

1. Deployment and headless Service configuration validation tests pass successfully.
2. Generated Deployment manifests carry valid OwnerReferences.
3. Multi-tenancy `TENANT_ID` is accurately propagated to container environment variables.
