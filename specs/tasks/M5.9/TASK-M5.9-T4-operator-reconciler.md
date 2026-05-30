# TASK-M5.9-T4: Operator Reconciler Customizer for Tiers Pod Generation

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.9-T4                                |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M5.9                                |
| Depends On    | TASK-M5.9-T3                                |

## Description

Updates the Operator's Reconciler service to load the mounted ConfigMap and map the CRD `spec.tier` to actual container CPU, memory, image pull policy, and probe configurations during pod deployment.

## Work Items

1. **RED Phase**:
   - Write unit tests in [reconcile_service_test.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/operator/application/service/reconcile_service_test.go) asserting that reconciling an Agent CRD with `spec.tier = "heavy"` results in a Pod spec containing "heavy" limits, while empty or missing tier falls back to "standard" specs.
   - Verify that the tests fail when run against the existing codebase.
2. **GREEN Phase**:
   - Write Go logic to load and parse the mounted ConfigMap JSON/YAML file in the Operator.
   - Update `ReconcileService` inside [reconcile_service.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/operator/application/service/reconcile_service.go) to extract `spec.tier` from the Agent CRD, look it up in the parsed tier map, and set the generated Pod container image tag, resources (requests/limits), and liveness/readiness probes accordingly.
   - Verify all tests compile and pass successfully.
3. **REFACTOR Phase**:
   - Optimize ConfigMap file parsing, cache parsed profiles on startup, and improve error wrappers.

## Acceptance Criteria

1. Reconciling an Agent CRD correctly applies container resources, pull policies, and probes from the tier ConfigMap, falling back cleanly to the default tier when not found.
