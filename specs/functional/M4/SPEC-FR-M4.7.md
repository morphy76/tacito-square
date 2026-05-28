# SPEC-FR-M4.7: Agent & Community Lifecycle Management REST API

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M4.7                                |
| Status        | IMPLEMENTED                                 |
| Milestone     | M4                                          |
| Component     | keeper, operator                            |
| Depends On    | SPEC-FR-M3.7, SPEC-FR-M4.1, SPEC-FR-M4.6     |
| Supersedes    | none                                        |

## Context

While Keeper provides full CRUD capabilities (Create, Retrieve, Update, Delete) for Agent templates and logical Communities, assigning an Agent to a Community automatically triggers CRD creation and runtime pod deployment (as specified in `SPEC-FR-M3.7`). 

However, operators and downstream consumers (such as the BFF or UI) require granular, independent controls to suspend (undeploy) and resume (deploy) running Agents/Communities without tearing down their database associations, and to fetch real-time K8s pod health status (e.g., Running, Error, Pending, Stopped).

This specification establishes the REST API endpoints and state machine transitions for executing Agent and Community lifecycle actions.

---

## Specification

### 1. Agent & Community Runtime State Machine
An assigned Agent Template possesses a runtime state managed by the Keeper, persisted in the database, and synced by the Operator:
*   **`stopped`**: Persisted database status indicating the record exists and is assigned to a community, but no active `TacitoAgent` CRD or Pod is provisioned in the cluster (manually suspended via `/undeploy`).
*   **`pending`**: CRD is submitted (via automatic assignment trigger or manual `/deploy`), but the K8s pod is still scheduling or pulling images.
*   **`running`**: Pod is active, healthy, and NATS listener threads are processing conversation streams.
*   **`error`**: Pod crashed, encountered OOM, or the model config failed validation at boot.

These statuses are added directly to the `model.AgentStatus` enum and are validated during domain verification.

A Community logical boundary supports the following statuses:
*   **`created`**: The community is defined but has not yet been deployed.
*   **`active`**: The community and all its assigned agents are deployed.
*   **`inactive`**: The community has been undeployed and all its assigned agents are stopped.
*   **`suspended`**: The community is paused (retaining resources but inactive).
*   **`terminated`**: The community has been deleted or retired.

The `inactive` status is added directly to `model.CommunityStatus` and is supported as a valid persistence state.

---

### 2. Agent Lifecycle Endpoints

#### A. Deploy Agent (Resume Suspension)
*   **Endpoint**: `POST /api/v1/agents/:agent_id/deploy`
*   **Behavior**:
    *   The Agent MUST be assigned to a Community (its `community_id` in the DB is non-nil).
    *   The dynamic Tenant context MUST match the Agent's `tenant_id` (returns `404 Not Found` if it belongs to a different tenant to prevent discovery attacks).
    *   The Keeper triggers the `CRDCoordinator` to build and submit the `TacitoAgent` CRD to the K8s API server.
    *   The Agent's state in the database transitions to `pending`.
    *   Publishes NATS `agent.provisioning.started` event. Once verified by the Operator/Sync, it publishes `agent.provisioning.completed` (or `agent.provisioning.failed` on failure).
*   **Responses**:
    *   `202 Accepted` on successful deployment trigger.
    *   `400 Bad Request` if the Agent is not assigned to a community.
    *   `409 Conflict` if the Agent is already in a `pending` or `running` state.

#### B. Undeploy Agent (Suspend Active)
*   **Endpoint**: `POST /api/v1/agents/:agent_id/undeploy`
*   **Behavior**:
    *   The Agent's current state MUST be `running`, `pending`, or `error`.
    *   The dynamic Tenant context MUST match.
    *   The Keeper invokes the `CRDCoordinator` to delete the corresponding `TacitoAgent` CRD from the K8s cluster.
    *   The deployed container executes its graceful shutdown sequence.
    *   The Agent's status transitions to `stopped` in the database.
    *   Publishes NATS `agent.provisioning.started` followed by `agent.provisioning.completed` on successful deletion.
*   **Responses**:
    *   `200 OK` on successful undeployment termination.
    *   `409 Conflict` if the Agent is already in a `stopped` state.

#### C. Get Agent Runtime Status
*   **Endpoint**: `GET /api/v1/agents/:agent_id/status`
*   **Behavior**:
    *   Queries real-time status parsed from the `TacitoAgent` CRD status subresource via the outbound `CRDCoordinator` port interface, avoiding direct transport coupling to K8s clients.
*   **Responses**:
    *   `200 OK` with JSON body:
        ```json
        {
          "agent_id": "uuid",
          "status": "running|pending|stopped|error",
          "message": "Pod healthy",
          "replicas": 1,
          "updated_at": "date-time"
        }
        ```

---

### 3. Community Lifecycle Endpoints

#### A. Deploy Community
*   **Endpoint**: `POST /api/v1/communities/:community_id/deploy`
*   **Behavior**:
    *   Retrieves all agents assigned to the community and invokes `Deploy Agent` logic **in parallel** via Go concurrent primitives (`golang.org/x/sync/errgroup`).
    *   Transitions community status to `active` in the database.
*   **Responses**:
    *   `202 Accepted` if all deployments are triggered successfully.
    *   `207 Multi-Status` if a subset of agent deployments fail to trigger. The response payload returns an array of details:
        ```json
        {
          "community_id": "uuid",
          "status": "partial_success",
          "agents": [
            { "agent_id": "uuid1", "status": "deployed", "error": "" },
            { "agent_id": "uuid2", "status": "failed", "error": "k8s connection refused" }
          ]
        }
        ```

#### B. Undeploy Community
*   **Endpoint**: `POST /api/v1/communities/:community_id/undeploy`
*   **Behavior**:
    *   Terminates all assigned agents within the community in parallel.
    *   Transitions community status to `inactive`.
*   **Responses**:
    *   `200 OK` if all deployments are terminated successfully.
    *   `207 Multi-Status` if a subset of agent terminations fail. The response payload returns an array of agent-level details.

#### C. Get Community Runtime Status
*   **Endpoint**: `GET /api/v1/communities/:community_id/status`
*   **Behavior**:
    *   Aggregates K8s and DB status of all agents assigned to this community.
*   **Responses**:
    *   `200 OK` with a detailed status breakdown per agent.

---

## Acceptance Criteria

1. **REST Endpoints Existence**: The 6 lifecycle routes exist in the Gin HTTP routing table.
2. **Dynamic Tenancy Validation**: All lifecycle commands validate the dynamic OIDC/JWT `tenant_id` context, returning `404 Not Found` if a tenant attempts actions on resources owned by another tenant.
3. **Bidirectional OpenAPI Verification**: Endpoints are fully documented in `api/openapi/openapi.json` and verified by our contract testing package.
4. **Hexagonal Architecture Separation**: HTTP adapters depend only on inbound use-case service port boundaries; GORM/pgx database logic and CRD orchestrations are isolated to outbound port drivers.

---

## Test Plan

### Automated Tests
1. **Unit Tests**:
    *   Verify Gin routes dispatch to the new lifecycle handler actions.
    *   Assert validation constraints (e.g. deploying an unassigned agent returns a `400` error).
2. **Integration Tests**:
    *   Assert database states update correctly during status transitions (`stopped`, `pending`, `running`, `error` for agents; `inactive` for communities).
    *   Assert concurrent parallel execution behavior and error aggregation (`207 Multi-Status`).
    *   Contract tests verify zero drift between route path registrations and the OpenAPI contract.

## Files Affected

- `[MODIFY] specs/functional/M4/SPEC-FR-M4.7.md`
- `[NEW] internal/keeper/application/ports/inbound/lifecycle_ports.go`
- `[MODIFY] internal/keeper/application/ports/outbound/crd_coordinator.go`
- `[NEW] internal/keeper/application/service/lifecycle_service.go`
- `[NEW] internal/keeper/adapters/inbound/http/lifecycle_handlers.go`
- `[MODIFY] internal/keeper/adapters/outbound/crd/crd_coordinator.go`
- `[MODIFY] internal/keeper/domain/model/agent.go`
- `[MODIFY] internal/keeper/domain/model/community.go`
- `[MODIFY] internal/keeper/bootstrap.go`
- `[MODIFY] api/openapi/openapi.json`
