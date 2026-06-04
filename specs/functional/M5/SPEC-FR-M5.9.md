# SPEC-FR-M5.9: Flexible Agent Runtime Tiers & Deployment Customization

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M5.9                                |
| Status        | ACCEPTED                                    |
| Milestone     | M5                                          |
| Component     | keeper, operator, deploy                    |
| Depends On    | SPEC-FR-M4.1, SPEC-FR-M4.3, SPEC-FR-M4.7   |
| Supersedes    | none                                        |

## Context

Deploying autonomous agent pods in a Kubernetes cluster requires operational flexibility. Different agents may require different container resource requests/limits, custom container image versions, pull policies, and customized liveness/readiness probe timings.

However, exposing these raw Kubernetes-level configurations directly to Keeper API end-users introduces several issues:
1. **Security Vulnerabilities:** End-users could mount malicious images, inject arbitrary environment variables, or request excessive resources (e.g., massive CPU/Memory bounds) causing Denial of Service (DoS) in the cluster.
2. **Tight Coupling:** The Keeper application layer would become tightly coupled to the underlying infrastructure specifications.

To solve this, this specification introduces **Flexible Agent Runtime Tiers**. The deployers define named tiers (such as `standard`, `heavy`, `gpu-optimized`) using Helm values at deploy time. End-users creating agents via the Keeper only select from these pre-defined logical tiers. The Operator's reconciliation loop maps the requested tier to actual Kubernetes resources, applying an **implicit default profile** (derived from the existing `agent.*` Helm values) whenever the requested tier is empty or unknown.

## Specification

### 1. Keeper Domain & Schema Changes
* The `Agent` domain model and PostgreSQL database schema MUST be updated to include a `tier` field (`varchar(50)`).
* `tier` is optional. If not supplied by the caller, it defaults to an empty string `""` in the database.
* The Keeper REST API `POST /api/v1/agents` and `PUT /api/v1/agents/:id` request bodies MUST accept an optional `tier` string parameter, nested under a `deployment` sub-object (e.g. `{ "deployment": { "tier": "heavy" } }`).
* The Keeper does **not** validate that the submitted tier name matches any known tier — it is a passthrough string. Tier resolution is entirely the Operator's responsibility.

### 2. Operator Custom Resource Definition (CRD) Alignment
* The `Agent` CRD spec struct (`AgentSpec`) MUST be updated to include a `tier` field (`spec.tier`).
* The existing `Resources *corev1.ResourceRequirements` field MUST be **removed** from `TacitoAgentSpec`, since resources are now fully determined by the Operator from the tier config. The Operator is the sole authority on pod resource sizing.
* When the Keeper's `CRDCoordinator` submits an `Agent` CRD to the Kubernetes API, it MUST populate `spec.tier` with the agent's persisted `tier` value.

### 3. Helm Values & ConfigMap Generation
* The Operator Helm chart (`tools/helm/tacito-square`) MUST define a configurable list of named tiers under `agentTiers.configs` in `values.yaml`. Each named entry overrides image, resources, pullPolicy, and probe timings.
* An **implicit default** profile exists, derived from the existing `agent.*` values in `values.yaml` (image, resources, probes). No explicit `agentTiers.default` key is needed; the implicit default is always available as the final fallback.
* The Helm chart MUST generate a Kubernetes `ConfigMap` named `ts-agent-tiers` in the operator's namespace, containing the YAML/JSON representation of all named tier configurations.

### 4. Operator Mount & Decoupled Reconciler
* The Operator deployment template MUST mount the `ts-agent-tiers` ConfigMap as a volume (e.g. at `/etc/tacito/tiers/tiers.yaml`).
* The Operator's `ReconcileService` MUST load and parse this tier mapping file **once at startup**. Tier configuration changes require an Operator pod restart (e.g. triggered automatically by a `helm upgrade`).
* When reconciling an `Agent` CRD:
  1. The Operator extracts the requested `tier` from `spec.tier`.
  2. It looks up the tier name in the loaded tier map.
  3. If the tier is empty, not found, or invalid, the Operator MUST fall back to the **implicit default profile** (derived from the `agent.*` Viper configuration keys, mirroring the existing startup behavior).
  4. The Operator applies the resolved profile's container `image` (repository, tag, pullPolicy), `resources` (requests and limits), and liveness/readiness `probes` timing parameters to the generated Pod template.

## Acceptance Criteria

1. **Logical Tier Decoupling:** Users cannot pass raw Kubernetes attributes (like CPU limits or image tags) via the Keeper API; they can only supply a logical `tier` string nested under `deployment`.
2. **Default Tier Fallback:** If an agent is defined with an empty `tier` or a non-existent tier name, the Operator successfully deploys the pod using the implicit default profile (as derived from the `agent.*` Helm values).
3. **Helm-Configured Resource Enforcement:** Pods deployed for agents with a known tier have their container CPU requests, CPU limits, Memory requests, Memory limits, image pull policy, and liveness/readiness probe timings populated exactly matching the Helm tier configuration.
4. **Decoupled Operator Config:** Swapping or updating tier specifications (via `helm upgrade`) takes effect on the next Operator pod restart without requiring Keeper restarts or database migrations.

## Test Plan

### Automated Tests
1. **Keeper Unit Tests (`agent_handlers.go`):**
   * Verify `POST /api/v1/agents` successfully binds and saves the optional `deployment.tier` field in the database.
   * Verify `PUT /api/v1/agents/:id` correctly updates the `tier` field.
2. **CRD Coordinator Integration Tests (`crd_coordinator.go`):**
   * Verify that the CRD submitted to Kubernetes has `spec.tier` correctly populated from the domain model.
   * Verify that an agent with an empty `tier` submits a CRD with an empty `spec.tier`.
3. **Operator Reconciler Unit Tests (`reconcile_service.go`):**
   * Mock the mounted tier ConfigMap configuration.
   * Verify that reconciling an Agent CRD with `spec.tier = "heavy"` generates a Pod template with "heavy" resources and image tags.
   * Verify that reconciling an Agent CRD with a non-existent or empty `spec.tier` falls back to the implicit default profile resources.

## Files Affected

### Keeper (Domain, REST, and DB)
* `[MODIFY] internal/keeper/domain/model/agent.go` — Add `Tier string` field.
* `[MODIFY] internal/keeper/adapters/inbound/http/agent_handlers.go` — Add optional `deployment.tier` to `CreateAgentRequest` and `UpdateAgentRequest`.
* `[NEW] deploy/postgres/migrations/00002_add_agent_tier.sql` — Goose migration to add `tier VARCHAR(50) NOT NULL DEFAULT ''` to the `agents` table.
* `[MODIFY] internal/keeper/adapters/outbound/postgres/agent_repository.go` — Include `tier` in all SELECT/INSERT/UPDATE queries.
* `[MODIFY] internal/keeper/openapi.json` — Document the new optional `deployment.tier` field on Agent create and update schemas.

### Keeper (CRD Outbound Adapter)
* `[MODIFY] internal/keeper/adapters/outbound/crd/crd_coordinator.go` — Populate `spec.tier` when constructing or patching the `TacitoAgent` CRD object.

### Kubernetes API (CRD Types)
* `[MODIFY] pkg/kubernetes/apis/tacito/v1alpha1/agent_types.go` — Add `Tier string` to `TacitoAgentSpec`; remove `Resources *corev1.ResourceRequirements` from `TacitoAgentSpec`.
* `[MODIFY] pkg/kubernetes/apis/tacito/v1alpha1/zz_generated.deepcopy.go` — Regenerate deep-copy after struct change (run `make generate`).

### Operator
* `[MODIFY] internal/operator/application/service/reconcile_service.go` — Load tier map from mounted file at startup; implement tier-to-profile resolution with implicit default fallback; apply profile to Pod template (image, resources, probes).
* `[NEW] tools/helm/tacito-square/templates/agent/configmap-tiers.yaml` — ConfigMap template that renders `agentTiers.configs` into `ts-agent-tiers`.
* `[MODIFY] tools/helm/tacito-square/templates/agent/operator-deployment.yaml` — Mount `ts-agent-tiers` ConfigMap as a volume at `/etc/tacito/tiers/`.
* `[MODIFY] tools/helm/tacito-square/values.yaml` — Add `agentTiers.configs` map with example named tiers.
