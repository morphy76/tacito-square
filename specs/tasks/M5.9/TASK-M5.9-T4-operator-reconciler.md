# TASK-M5.9-T4: Operator Reconciler Tier Resolution for Pod Generation

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.9-T4                                |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M5.9                                |
| Depends On    | TASK-M5.9-T3                                |

## Description

Updates the Operator's `ReconcileService` to load the mounted `ts-agent-tiers` ConfigMap once at startup, resolve a pod profile from `spec.tier`, and apply that profile's image, resource constraints, and probe timings to the generated Pod template. Falls back to the implicit default profile (from existing `agent.*` Viper settings) when the tier is empty or unmatched.

## Work Items

1. **RED Phase**:
   - Write unit tests in [reconcile_service_test.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/operator/application/service/reconcile_service_test.go) asserting:
     - Reconciling an Agent CRD with `spec.tier = "heavy"` produces a Pod with "heavy" resource limits and image.
     - Reconciling with an empty or unknown `spec.tier` produces a Pod using the implicit default profile (matching existing `agent.*` Viper values).
   - Verify that the tests fail against the existing codebase.
2. **GREEN Phase**:
   - Define a `TierProfile` struct (image, pullPolicy, resources, livenessProbe timings, readinessProbe timings) and a `TierMap` loader that reads and parses `/etc/tacito/tiers/tiers.yaml` at process startup.
   - Update `NewReconcileAgentService` constructor in [reconcile_service.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/operator/application/service/reconcile_service.go) to accept and store the loaded `TierMap`.
   - In `BuildDeployment`: remove the `agent.Spec.Resources` usage (field no longer exists); instead resolve the tier profile from the `TierMap` using `agent.Spec.Tier`, falling back to the implicit default profile built from the existing `agent.*` Viper keys (`agent.image`, `agent.resources.*`, etc.).
   - Apply the resolved profile's `image`, `pullPolicy`, `resources`, and probe timing overrides to the Pod container spec.
   - Verify all tests compile and pass.
3. **REFACTOR Phase**:
   - Cache the parsed `TierMap` and implicit default profile at construction time, not per-reconcile call.
   - Improve error wrapping around tier file parse failures with a clear startup error log.

## Acceptance Criteria

1. Reconciling an Agent CRD with a known `spec.tier` applies that tier's resources, image, and probes to the Pod template.
2. Reconciling with an empty or unknown `spec.tier` falls back cleanly to the implicit default profile without error.
3. The `ReconcileService` does not read `spec.Resources` from the CRD (field removed in T2).
