# SPEC-FR-M4.7: Agent & Community Lifecycle Management REST API

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M4.7                                |
| Status        | DRAFT                                       |
| Milestone     | M4                                          |
| Component     | keeper, operator                            |
| Depends On    | SPEC-FR-M3.7, SPEC-FR-M4.1, SPEC-FR-M4.6     |
| Supersedes    | none                                        |

## Context

While Keeper provides full CRUD capabilities (Create, Retrieve, Update, Delete) for Agent templates and logical Communities, it currently lacks explicit lifecycle control endpoints to programmatically **deploy**, **undeploy**, and query the **runtime status** of existing assigned Agents and Communities.

Currently, assigning an Agent to a Community automatically triggers CRD creation under Milestone 3 boundaries. However, operators and downstream consumers (such as the BFF or UI) require granular, independent controls to suspend (undeploy) and resume (deploy) running Agents/Communities without tearing down their database associations, and to fetch real-time K8s pod health status (e.g. Running, Error, Pending, Stopped).

This specification establishes the REST API endpoints and state machine transitions for executing Agent and Community lifecycle actions.

---

## Specification

### 1. Agent Runtime State Machine
An assigned Agent Template possesses a runtime state managed by the Keeper and synced by the Operator:
- **`stopped`**: GORM database record exists and is assigned to a community, but no active `TacitoAgent` CRD or Pod is provisioned in the cluster.
- **`pending`**: CRD is submitted, but the K8s pod is still scheduling or pulling images.
- **`running`**: Pod is active, healthy, and NATS listener threads are processing conversation streams.
- **`error`**: Pod crashed, encountered OOM, or the model config failed validation at boot.

### 2. Agent Lifecycle endpoints

#### A. Deploy Agent
- **Endpoint**: `POST /api/v1/agents/:agent_id/deploy`
- **Behavior**:
  - The Agent MUST be assigned to a Community (its `community_id` in the DB is non-nil).
  - The dynamic Tenant context MUST match the Agent's `tenant_id`.
  - The Keeper triggers the `CRDCoordinator` to build and submit the `TacitoAgent` CRD to the K8s API server.
  - The Agent's state in the database transitions to `pending` or `running` (once verified).
- **Responses**:
  - `202 Accepted` on successful deployment trigger.
  - `400 Bad Request` if the Agent is not assigned to a community.
  - `409 Conflict` if the Agent is already deployed/active.

#### B. Undeploy Agent
- **Endpoint**: `POST /api/v1/agents/:agent_id/undeploy`
- **Behavior**:
  - The Agent's current state MUST be `running`, `pending`, or `error`.
  - The Keeper invokes the `CRDCoordinator` to delete the corresponding `TacitoAgent` CRD from the K8s cluster.
  - The deployed container executes its graceful shutdown sequence (completing active requests within a timeout).
  - The Agent's status transitions to `stopped` in the database.
- **Responses**:
  - `200 OK` on successful undeployment termination.
  - `409 Conflict` if the Agent is already undeployed/stopped.

#### C. Get Agent Runtime Status
- **Endpoint**: `GET /api/v1/agents/:agent_id/status`
- **Behavior**:
  - Returns real-time status parsed from the `TacitoAgent` CRD status subresource or corresponding pod conditions (queried via `client-go` using OID/Namespace context).
- **Responses**:
  - `200 OK` with JSON body:
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
- **Endpoint**: `POST /api/v1/communities/:community_id/deploy`
- **Behavior**:
  - Deploys the community logical resources and deploys **all assigned agents** in parallel.
  - Transitions community status to `active`.
- **Responses**:
  - `202 Accepted`.

#### B. Undeploy Community
- **Endpoint**: `POST /api/v1/communities/:community_id/undeploy`
- **Behavior**:
  - Undeploys/terminates all assigned agents within the community in parallel.
  - Transitions community status to `inactive`.
- **Responses**:
  - `200 OK`.

#### C. Get Community Runtime Status
- **Endpoint**: `GET /api/v1/communities/:community_id/status`
- **Behavior**:
  - Aggregates status of all agents assigned to this community.
- **Responses**:
  - `200 OK` with status breakdown per agent.

---

## Acceptance Criteria

1. **REST Endpoints Existence**: The 6 lifecycle routes exist unconditionally in the Gin HTTP routing table.
2. **Dynamic Tenancy Validation**: All lifecycle commands validate the dynamic OIDC/JWT `tenant_id` context, returning `404 Not Found` if a tenant attempts actions on resources owned by another tenant.
3. **Bidirectional OpenAPI Verification**: Endpoints are fully documented in `api/openapi/openapi.json` and verified by our contract testing package.
4. **Hexagonal Architecture Separation**: HTTP adapters depend only on inbound use-case service port boundaries; GORM database logic is isolated to outbound port drivers.

---

## Test Plan

### Automated Tests
1. **Unit Tests**:
   - Verify Gin routes dispatch to the new lifecycle handler actions.
   - Assert validation constraints (e.g. deploying an unassigned agent returns a `400` error).
2. **Integration Tests**:
   - Assert GORM database states update correctly during status transitions.
   - Contract tests verify zero drift between route path registrations and the OpenAPI contract.

## Files Affected

- `[NEW] specs/functional/M4/SPEC-FR-M4.7.md`
- `[NEW] internal/keeper/application/ports/inbound/lifecycle_ports.go`
- `[MODIFY] internal/keeper/adapters/inbound/http/lifecycle_handlers.go`
- `[MODIFY] internal/keeper/bootstrap.go`
- `[MODIFY] api/openapi/openapi.json`
