---
trigger: glob
globs: ["**/*.{go,ts,yaml,Dockerfile}"]
description: Kubernetes deployment requirements, Kubebuilder operator reconcilers, HPA autoscaling, health probes, and distroless container configurations.
---

# Kubernetes Best Practices Guidelines

This rule enforces Kubernetes-native configuration, Kubebuilder controller patterns, autoscaling design, deployment health validation, and container base constraints.

## 1. Kubernetes Operator & Controller Patterns (`sigs.k8s.io/controller-runtime`)

All controllers and reconcilers (e.g. `cmd/operator`, `operator/`) must adhere to cloud-native reconciliation semantics:

- **Idempotence**: The `Reconcile(ctx context.Context, req ctrl.Request)` loop MUST be strictly idempotent. Running reconcile repeatedly against the same state must produce identical results without duplicating pods, secrets, or services.
- **Level-Triggered State**: Reconcilers must reconcile toward the desired state defined in the CRD spec (`Agent`, `Community`), not assume sequential step transitions.
- **Status Condition Updates**: Use standard Kubernetes condition arrays (`metav1.Condition`) to report state (`Ready`, `Reconciling`, `Error`) with clear `Reason` and `Message`.
- **Finalizer Hygiene**: When managing external resources (e.g. stateful storage, external registrations), register a finalizer and ensure cleanup executes before removing the finalizer string.
- **Error Requeue Strategy**: Return transient errors with exponential backoff (`ctrl.Result{RequeueAfter: ...}`). Return non-recoverable domain/validation errors without requeue to avoid infinite hot loops.
- **Pure Reconciler Layer**: Reconcilers must orchestrate Kubernetes manifests, delegating complex domain logic and config compilation to application services.

## 2. Horizontal Pod Autoscaling (SPEC-NFR-HPA)

Every deployable microservice (e.g., `keeper`, `agent`, `bff`) must support automated scaling configurations:

- **HPA Configurations:** Every deployable artifact MUST have an HPA configuration with load-factor-based scaling. HPA templates must be included in Helm charts (`tools/helm/`) with configurable `minReplicas`, `maxReplicas`, and metrics.
- **Component Scaling Metrics:**
  - **Agent Component:** 
    - Primary Metric: `active_threads` (custom Prometheus metric)
    - Secondary Metric: CPU utilization
    - Replica Limits: Min 1, Max 10 (respecting the community `max_agents` ceiling quota)
  - **Keeper Component:**
    - Primary Metric: HTTP request rate
    - Secondary Metric: CPU utilization
    - Replica Limits: Min 2, Max 5
  - **BFF Component:**
    - Primary Metric: HTTP request rate
    - Secondary Metric: CPU utilization
    - Replica Limits: Min 2, Max 5
- **Prometheus Metric Exporting:** All custom scaling metrics (e.g., `active_threads`) must be exported in Prometheus exposition format and consumed by the HPA adapter (`prometheus-adapter` or KEDA).
- **Idle agent scale-to-zero:** Scale-to-zero should be supported for idle agents (this capability must be configurable and defaulted to `disabled`).

## 3. Dependency-Aware Health Probes (SPEC-NFR-HEALTH)

All network-exposed Go/TypeScript processes must implement and expose `/healthz` and `/readyz` probes:

- **Liveness Probe (`/healthz`):** 
  - Purpose: Verify if the process is alive.
  - Checks: No dependency checks. Always return `200 OK` (JSON format) if process is up.
  - Logging: Only emit logs for failures (to avoid log noise) and the first success after a failure.
- **Readiness Probe (`/readyz`):**
  - Purpose: Verify that the process is ready to handle traffic by checking its architectural backing services.
  - Checks: Check ALL downstream architectural dependencies **in parallel** with a configurable timeout.
  - Outbound calls: Do not check HTTP endpoints (to avoid cascading readiness checks), unless infrastructural (e.g., Redis, NATS). Use Go's `Ping` or connection check functions.
  - **Per-Component Dependency Checks:**
    - **Keeper:** PostgreSQL ping, NATS connection, Redis ping, Cache Redis ping.
    - **Agent:** NATS connection, Redis ping, Cache Redis ping, Qdrant ping.
  - Response: 
    - If healthy, return `200 OK` (JSON format).
    - If any dependency is unhealthy, return `503 Service Unavailable` with details and errors per-dependency (JSON format).
  - Logging: Only log readiness failures and first recovery success.

## 4. Container Images (SPEC-NFR-STACK)

Ensure all microservice deployment containers utilize secure, lightweight base images:

- **Base Image Constraint:** All Dockerfiles MUST use Google Distroless images as their runtime base:
  - Base: `gcr.io/distroless/base-nossl-debian13`
- **Build Stage:** Use a multi-stage Docker build structure (compile in a full Go SDK builder image, and copy only the compiled binary to the distroless base image).

---

## Developer Checklists & Verifications

- [ ] Is my controller reconciler idempotent and non-blocking?
- [ ] Does my Dockerfile use `gcr.io/distroless/base-nossl-debian13` as the runtime stage?
- [ ] Are `/healthz` and `/readyz` endpoints exposed?
- [ ] Does `/readyz` perform dependency checks in parallel?
- [ ] Is there an HPA template in the Helm chart for this service?
