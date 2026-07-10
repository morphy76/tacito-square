# SPEC-FR-M6.5.1: Agent Role as Community Assignment Behavior

| Field       | Value |
|-------------|-------|
| ID          | SPEC-FR-M6.5.1 |
| Status      | DRAFT |
| Milestone   | M6.5 |
| Component   | keeper, operator |
| Depends On  | SPEC-FR-M6.1 |
| Supersedes  | SPEC-FR-M6.1 (role column clause only) |

## Context

Under DDD, agent role is a **behavior** exhibited when an agent is placed in a community with a given topology — not an intrinsic property of the Agent aggregate. The current keeper implementation embeds a `Role string` field on the `Agent` struct in `internal/keeper/domain/model/agent.go` (values: `hub`, `spoke`). This is a DDD violation: the Agent should be role-agnostic; the role is determined by the community assignment. This spec corrects the model and extends the role enum with `standalone` for single-agent topology communities.

## Specification

1. **Remove** the `Role string` field and its validation from the `Agent` aggregate in `internal/keeper/domain/model/agent.go`.

2. **Define** a new `AgentRole` type in the keeper domain:
   ```go
   type AgentRole string
   const (
       AgentRoleStandalone AgentRole = "standalone"
       AgentRoleHub        AgentRole = "hub"
       AgentRoleSpoke      AgentRole = "spoke"
   )
   ```

3. **Create** a new `CommunityAssignment` aggregate in `internal/keeper/domain/model/community_assignment.go`:
   ```go
   type CommunityAssignment struct {
       CommunityID uuid.UUID
       AgentID     uuid.UUID
       Role        AgentRole
       InformedAt  *time.Time // nil = stale/unreconciled
       AssignedAt  time.Time
   }
   ```
   This replaces the `Agent.CommunityID` + `Agent.Role` fields as the assignment record.

4. **Assignment validation rules** (enforced by the Keeper assignment domain service):
   - `single-agent` topology → role must be `standalone`; maximum one agent per community.
   - `hub-spoke` topology → exactly one `hub` allowed; multiple `spoke` agents allowed.
   - Role is **explicitly provided** by the API caller when assigning an agent to a hub-spoke community. For single-agent topology it is automatically set to `standalone`.

5. **DB migration**: Create `community_assignments(community_id UUID, agent_id UUID, role TEXT, informed_at TIMESTAMPTZ, assigned_at TIMESTAMPTZ, PRIMARY KEY (community_id, agent_id))`. Temporarily retain `agents.role` as a deprecated column (populated by migration trigger for backward compat); schedule removal in M6.5 cleanup.

6. **Operator**: The `TacitoAgent` CRD `spec.role` field is populated by Keeper from the assignment record at reconciliation time. The Operator sets `TS_AGENT_ROLE` env var from `spec.role`.

## Acceptance Criteria

1. Creating an agent via POST /api/v1/agents does not accept or return a `role` field.
2. Assigning an agent to a `single-agent` community automatically sets role = `standalone` on the assignment record.
3. Assigning an agent to a `hub-spoke` community requires an explicit `role` field (`hub` or `spoke`).
4. A second `hub` assignment to an existing hub-spoke community is rejected with `409 Conflict`.
5. Unassigning an agent (`DELETE /api/v1/communities/{id}/agents/{agent_id}`) removes the assignment record.
6. The Operator sets `TS_AGENT_ROLE` from the assignment's role value on pod creation.

## Test Plan

- **Unit**: `CommunityAssignment.Validate()` enforces topology/role rules.
- **Unit**: Assignment service rejects duplicate hub with domain error.
- **Integration**: Keeper API end-to-end for assignment and unassignment.
- **Integration**: Operator reconciler reads role from CRD spec and sets `TS_AGENT_ROLE`.

## API Contract

```
POST   /api/v1/communities/{id}/agents
       Body: { "agent_id": "uuid", "role": "hub|spoke" }  // role omitted for single-agent topology
       Response 200: { "agent_id": "uuid", "role": "...", "assigned_at": "..." }
       Response 409: { "error": "a hub agent is already assigned to this community" }

DELETE /api/v1/communities/{id}/agents/{agent_id}
       Response 204

GET    /api/v1/communities/{id}/agents
       Response 200: [{ "agent_id": "uuid", "role": "...", "assigned_at": "...", "informed_at": "..." }]
```

## Files Affected

- `internal/keeper/domain/model/agent.go` [MODIFY] — remove `Role` field and validation
- `internal/keeper/domain/model/community_assignment.go` [NEW] — `AgentRole` type + `CommunityAssignment` aggregate
- `internal/keeper/application/ports/outbound/community_assignment_repository.go` [NEW]
- `internal/keeper/application/service/community_service.go` [MODIFY] — assignment validation logic
- `internal/keeper/adapters/inbound/http/community_handler.go` [MODIFY] — assignment endpoints
- `pkg/kubernetes/apis/tacito/v1alpha1/tacitoagent_types.go` [MODIFY] — `spec.role` from assignment record
- DB migration [NEW] — `community_assignments` table
