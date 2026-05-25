# SPEC-FR-M4.3: Reconciliation Controller

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M4.3                                |
| Status        | ACCEPTED                                    |
| Milestone     | M4                                          |
| Component     | operator                                    |
| Depends On    | SPEC-FR-M4.1                                |
| Supersedes    | none                                        |

## Context

The operator must watch `TacitoAgent` CRDs and reconcile them into Kubernetes Deployments and headless Services. This is the core control loop that translates declarative agent definitions into running pods in the cluster.

## Specification

1. **Controller Framework & Hexagonal Structure**:
   * The operator MUST implement a `controller-runtime` reconciler watching `TacitoAgent` custom resources.
   * The reconciler MUST be structured as an **Inbound Driving Adapter** under `internal/operator/adapters/inbound/reconciler.go`.
   * Core deployment orchestration business logic MUST reside in the application use-case layer (`internal/operator/application/service/`).

2. **Reconciliation Target (Deployment & Service Creation)**:
   * **Deployment**: On creation, the reconciler MUST spawn a Deployment with 1 container utilizing the Agent Docker image (`tacito-square/agent`).
   * **Headless Service**: The reconciler MUST spawn a matching headless Service (ClusterIP: `None`) pointing to the agent container's port `8081` for peer-to-peer and community message-routing identities.

3. **Multi-Tenancy & Base Configuration Injection**:
   * **Tenant ID**: The reconciler MUST read `spec.tenantId` from the `TacitoAgent` resource and inject it as the `TENANT_ID` environment variable into the agent Pod.
   * **Infrastructure Settings (Option A)**: The Operator loads base connection settings at startup (via Viper) and injects them as standard environment variables into every spawned agent Pod:
     * `TS_AGENT_NATS_URL` (NATS connection endpoint)
     * `TS_AGENT_REDIS_URL` (Redis connection endpoint)
     * `TS_AGENT_QDRANT_URL` (Qdrant connection endpoint)
     * `TS_AGENT_OTEL_ENDPOINT` (OpenTelemetry collector endpoint)
     * `TS_AGENT_S3_ENDPOINT` (S3 object storage endpoint)
   * **Agent-Specific Parameters**:
     * `TS_AGENT_NAME` (from `spec.agentName`)
     * `TS_AGENT_COMMUNITY_REF` (from `spec.communityRef`)
     * `TS_AGENT_BRAIN_MODEL` (from `spec.llmConfig.model`)
     * `TS_AGENT_BRAIN_TEMPERATURE` (from `spec.llmConfig.temperature`)
     * `TS_AGENT_BRAIN_MAX_TOKENS` (from `spec.llmConfig.maxTokens`)
     * `TS_AGENT_SYSTEM_PROMPT` (from `spec.systemPrompt`)

4. **Resource Constraints & Limits Mapping**:
   * The container's CPU and Memory requests/limits MUST be mapped directly from `spec.resources` in the `TacitoAgent` resource to the Deployment manifest.

5. **Resource Lifecycle & Cascade Cleanup**:
   * The reconciler MUST set the `OwnerReference` of both the Deployment and the Service to the parent `TacitoAgent` resource. This ensures automatic, native Kubernetes cascade garbage collection on agent deletion.
   * Prepare a finalizer (`tacito.square.io/agent-finalizer`) to handle downstream external notification tasks (such as notifying the Keeper component of de-provisioning) if required.

6. **Status Reporting & Phase Management**:
   * The reconciler MUST keep `status.phase` in sync with the backing Deployment availability:
     * `Pending` if Pods are creating or image pull is in progress.
     * `Running` if at least 1 Pod replica is healthy and running.
     * `Idle` if replicas are scaled to `0`.
     * `Terminated` if the resource is undergoing deletion.
   * Standard Kubernetes conditions (`Available`, `Progressing`, `Failed`) MUST be written to `status.conditions` using standard metav1 Condition writers.

7. **Operator Observability & Custom Metrics**:
   * The reconciler MUST export the following custom metrics in standard Prometheus exposition format on `GET /metrics`:
     * `tacito_operator_reconciliation_total` (Counter: partitioned by status="success|error")
     * `tacito_operator_reconciliation_duration_seconds` (Histogram: tracking latency)
     * `tacito_operator_active_agents` (Gauge: partitioned by phase="Pending|Running|Idle|Terminated")

## Acceptance Criteria

1. **Deployment Matching**:
   * Creating a `TacitoAgent` resource MUST result in the automatic creation of a Deployment and a headless Service named after the custom resource.
2. **Configuration Completeness**:
   * The created Deployment container MUST have all environment variables (both base infrastructure parameters and agent-specific spec parameters) set correctly, and matches `spec.tenantId` in `TENANT_ID`.
3. **Status Sync Accuracy**:
   * If the Deployment has 0 active replicas, `status.phase` MUST reflect `Idle`. If the Deployment becomes available, `status.phase` MUST reflect `Running`.
4. **Observability Exporter**:
   * Successful reconciliations MUST increment `tacito_operator_reconciliation_total{status="success"}`. Failed reconciliations (e.g. invalid configurations or namespace conflicts) MUST increment `status="error"`.
5. **Cascade Deletion Reliability**:
   * Deleting the `TacitoAgent` resource MUST result in automatic deletion of the associated Deployment and Service by the Kubernetes garbage collector without orphan processes.

## Test Plan

### Automated Tests
* **Envtest Reconciliation Suite**:
  * Implement integration tests under `internal/operator/adapters/inbound/reconciler_test.go` using `sigs.k8s.io/controller-runtime/pkg/envtest`.
  * Verify that creating a `TacitoAgent` CR successfully creates a Deployment and headless Service.
  * Verify that updating `TacitoAgent` fields updates the underlying Deployment environment variables.
  * Mock standard Deployment replica updates and verify that `status.phase` transition rules evaluate correctly.
* **Makefile Integration**:
  * Execute via `make test-operator`.

### Manual Verification
* Deploy the Operator to a local Kind/Minikube cluster, apply a sample `TacitoAgent` CR, and inspect the resulting Pod's environment variables (`kubectl env pod/<pod-name>`) and Prometheus metrics (`curl localhost:8082/metrics`).

## Files Affected

* `internal/operator/adapters/inbound/reconciler.go` [NEW]
* `internal/operator/adapters/inbound/reconciler_test.go` [NEW]
* `internal/operator/application/service/reconcile_service.go` [NEW]
* `internal/operator/bootstrap.go` (Register the reconciler with the manager setup)

