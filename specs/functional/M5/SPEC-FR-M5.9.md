# SPEC-FR-M5.9: Flexible Agent Runtime Tiers & Deployment Customization

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M5.9                                |
| Status        | ACCEPTED                                    |
| Milestone     | M5                                          |
| Component     | keeper, operator, deploy                    |
| Depends On    | SPEC-FR-M4.1, SPEC-FR-M4.3, SPEC-FR-M4.7    |
| Supersedes    | none                                        |

## Context

Deploying autonomous agent pods in a Kubernetes cluster requires operational flexibility. Different agents may require different container resource requests/limits, custom container image versions, pull policies, and customized liveness/readiness probe timings. 

However, exposing these raw Kubernetes-level configurations directly to Keeper API end-users introduces several issues:
1. **Security Vulnerabilities:** End-users could mount malicious images, inject arbitrary environment variables, or request excessive resources (e.g., massive CPU/Memory bounds) causing Denial of Service (DoS) in the cluster.
2. **Tight Coupling:** The Keeper application layer would become tightly coupled to the underlying infrastructure specifications.

To solve this, this specification introduces **Flexible Agent Runtime Tiers**. The deployers define named tiers (such as `standard`, `heavy`, `gpu-optimized`) using Helm values at deploy time. End-users creating agents via the Keeper only select from these pre-defined logical tiers. The Operator's reconciliation loop maps the requested tier to actual Kubernetes resources, falling back to a pre-defined default tier if the requested tier is missing or invalid.

## Specification

### 1. Keeper Domain & Schema Changes
* The `Agent` domain model and PostgreSQL database schema MUST be updated to include a `tier` field (`varchar(50)`).
* `tier` is optional. If not supplied, it defaults to an empty string `""` in the database.
* The Keeper REST API `POST /api/v1/agents` and `PUT /api/v1/agents/:id` request bodies MUST accept an optional `tier` string parameter.

### 2. Operator Custom Resource Definition (CRD) Alignment
* The `Agent` Custom Resource Definition (CRD) spec struct (`AgentSpec`) MUST be updated to include a `tier` field (`spec.tier`).
* When the Keeper's `CRDCoordinator` submits an `Agent` CRD to the Kubernetes API, it MUST populate the `spec.tier` field with the agent's database `tier` value.

### 3. Helm Values & ConfigMap Generation
* The Operator Helm chart (`deploy/helm/tacito-square`) MUST define a configurable list of tiers and a default fallback key in `values.yaml` (e.g. `agentTiers.default` and `agentTiers.configs`).
* The Helm chart MUST generate a Kubernetes `ConfigMap` named `ts-agent-tiers` in the operator's namespace containing the YAML/JSON representation of the tier configurations.

### 4. Operator Mount & Decoupled Reconciler
* The Operator deployment template MUST mount the `ts-agent-tiers` ConfigMap as a volume (e.g. at `/etc/tacito/tiers/`).
* The Operator's reconciliation service (`ReconcileService`) MUST load and parse this tier mapping file on startup (or watch for changes dynamically).
* When reconciling an `Agent` CRD:
  1. The Operator extracts the requested `tier` from `spec.tier`.
  2. It looks up the corresponding configuration profile from the tier map.
  3. If the requested tier is empty, not found, or invalid, the Operator MUST fall back to the default profile configuration defined by the default key.
  4. The Operator applies the selected profile's container `image` (repository, tag, pullPolicy), `resources` (requests and limits), and liveness/readiness `probes` parameters to the generated Pod template.

## Acceptance Criteria

1. **Logical Tier Decoupling:** Users cannot pass raw Kubernetes attributes (like CPU limit or image tag) via the Keeper API; they can only supply a logical `tier` string.
2. **Default Tier Fallback:** If an agent is defined with an empty `tier` or a non-existent tier name, the Operator successfully deploys the pod using the `default` tier specs defined in Helm.
3. **Helm-Configured Resource Enforcement:** Pods deployed for agents assigned to a community have their container CPU requests, CPU limits, Memory requests, Memory limits, image pull policies, and liveness/readiness probes populated exactly matching the mapped Helm tier configuration.
4. **Decoupled Operator Config:** Swapping or updating tier specifications in the ConfigMap (via Helm upgrade) immediately takes effect on the next operator reconciliation loop without requiring Keeper restarts or database migrations.

## Test Plan

### Automated Tests
1. **Keeper Unit Tests:**
   * Verify `POST /api/v1/agents` successfully binds and saves the optional `tier` field in the database.
2. **CRD Coordinator Integration Tests:**
   * Verify that the generated CRD submitted to Kubernetes has the correct `spec.tier` populated.
3. **Operator Reconciler Unit Tests:**
   * Mock the mounted tier ConfigMap configuration.
   * Verify that reconciling an Agent CRD with `spec.tier = "heavy"` generates a Pod template with "heavy" resources and image tags.
   * Verify that reconciling an Agent CRD with a non-existent or empty `spec.tier` falls back to the "standard" default tier resources.

## Files Affected

### Keeper (Domain, REST, and DB)
* `[MODIFY] internal/keeper/domain/model/agent.go` — Add `Tier` field.
* `[MODIFY] internal/keeper/adapters/inbound/http/agent_handlers.go` — Update REST request binding.
* `[MODIFY] internal/keeper/adapters/outbound/postgres/migrations/` — Database schema update.
* `[MODIFY] internal/keeper/openapi.json` — Document the new optional `tier` field on the Agent schema.

### Kubernetes API
* `[MODIFY] pkg/kubernetes/apis/tacito/v1alpha1/agent_types.go` — Add `Tier` to the `AgentSpec` struct.

### Operator
* `[MODIFY] internal/operator/application/service/reconcile_service.go` — Integrate ConfigMap loading and Pod template customizer.
* `[NEW] deploy/helm/tacito-square/templates/operator/configmap-tiers.yaml` — ConfigMap generator template.
