# SPEC-FR-M3.7: Agent-Community Assignment

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M3.7                                |
| Status        | IMPLEMENTED                                 |
| Milestone     | M3                                          |
| Component     | keeper                                      |
| Depends On    | SPEC-FR-M3.5, SPEC-FR-M3.6                  |
| Supersedes    | none                                        |

## Context

Agents are defined as templates, but their runnable containerized instances in the cluster are bound to a specific **Community** (the logical messaging and isolation boundary). This specification covers the assignment and unassignment lifecycle of agent definitions to communities, which triggers CRD creation/deletion and coordinates pod orchestration via the operator.

Under Milestone 4, this assignment/unassignment lifecycle is seamlessly integrated with the operator deployment controls, allowing automatic deployment upon assignment and manual overrides using explicit lifecycle APIs.

---

## Specification

### 1. The One-Community Constraint
- An agent template belongs to **exactly one** community at any given time.
- If an agent template is already assigned to a community, attempting to assign it to a different community **MUST fail with a 409 Conflict** error. The user/system must explicitly unassign the agent first.

### 2. Assignment API Endpoint
- **Path**: `POST /api/v1/communities/:community_id/agents/:agent_id`
- **Validation**:
  - Both the Community and the Agent MUST exist and belong to the authenticated tenant.
  - The community's status MUST be `active` or `created`.
  - The agent's current `community_id` MUST be null.
- **Action**:
  - Set the Agent's `community_id` in the database to `community_id`.
  - Trigger the submission of the `TacitoAgent` CRD to Kubernetes to automatically provision the runtime container (per `SPEC-FR-M4.6`).
  - Update the Agent's status in the database to `pending`.
- **Manual Overrides**:
  - Once assigned, the agent's running instance can be manually suspended (undeployed) or resumed (deployed) using the lifecycle endpoints specified in `SPEC-FR-M4.7`, transitioning status between `stopped` and `pending` without removing the community association.

### 3. Unassignment API Endpoint
- **Path**: `DELETE /api/v1/communities/:community_id/agents/:agent_id`
- **Validation**:
  - The agent's `community_id` MUST currently match the `:community_id` in the route parameter.
- **Action**:
  - **Graceful Termination Sequence**: Unassigning the agent triggers a teardown of the corresponding `TacitoAgent` CRD from the cluster (per `SPEC-FR-M4.6` / `SPEC-FR-M4.7`):
    1. Terminate current active conversation threads without accepting any new incoming messages.
    2. Fall back to a configurable hard timeout (e.g., 30 seconds), after which the container is forcibly terminated.
  - Set the Agent's `community_id` to `NULL`.
  - Update the Agent's status back to `defined`.

---

## Acceptance Criteria

1. **Clean State Management**:
   - Assigning a valid unassigned agent to a valid community successfully updates `community_id` in the DB, triggers K8s CRD coordinator submission, and transitions its status to `pending`.
   - Re-assigning an already assigned agent directly returns `409 Conflict`.

2. **Security & Multi-Tenancy**:
   - Attempting to assign an agent of `Tenant A` to a community of `Tenant B` must fail with `404 Not Found` (to prevent leaking the existence of other tenants' entities).

3. **CRD Triggers**:
   - Calling the assignment endpoints invokes the CRD coordinator to create/delete custom resources.

---

## Test Plan

### 1. Integration Tests
- **Assignment Success**: Verify status transitions to `pending` and community ID updates in GORM repository.
- **Assignment Guard**: Try assigning an already assigned agent and assert a 409 Conflict response.
- **Unassignment Lifecycle**: Unassign an agent, check that `community_id` becomes `NULL` and status is set back to `defined`.
- **Tenant Validation**: Attempt cross-tenant assignments and assert failures.
