# SPEC-FR-M6.5.1: Agent Role as Community Assignment Behavior

| Field       | Value |
|-------------|-------|
| ID          | SPEC-FR-M6.5.1 |
| Status      | VERIFIED |
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
       TenantID    string
       Role        AgentRole
       InformedAt  *time.Time // nil = stale/unreconciled
       AssignedAt  time.Time
   }
   ```
   This replaces the `Agent.CommunityID` + `Agent.Role` fields as the assignment record.

4. **Assignment validation rules** (enforced by the Keeper assignment domain service):
   - `single-agent` topology → role must be `standalone`; maximum one agent per community.
   - `hub-spoke` topology → exactly one `hub` allowed; multiple `spoke` agents allowed.
   - Role is **explicitly provided** by the API caller when assigning an agent to a hub-spoke community (`hub` or `spoke`). For single-agent topology it is automatically set to `standalone`, ignoring any caller-supplied role value.
   - Validation is performed by the assignment use-case service, which loads the community topology before persisting the record. Invalid topology/role combinations (e.g. a second `hub`, or `hub` in a single-agent community) are rejected with a domain error that maps to `409 Conflict`.

5. **Update** the `AssignmentUseCase` inbound port signature:
   ```go
   type AssignmentUseCase interface {
       Assign(ctx context.Context, communityID uuid.UUID, agentID uuid.UUID, role model.AgentRole) error
       Unassign(ctx context.Context, communityID uuid.UUID, agentID uuid.UUID) error
       ListByCommunity(ctx context.Context, communityID uuid.UUID) ([]*model.CommunityAssignment, error)
   }
   ```
   The HTTP handler supplies the caller-provided role (or an empty value for single-agent requests); the service resolves and validates it against the community topology.

6. **Define** the `CommunityAssignmentRepository` outbound port in `internal/keeper/application/ports/outbound/community_assignment_repository.go`:
   ```go
   type CommunityAssignmentRepository interface {
       Create(ctx context.Context, assignment *model.CommunityAssignment) error
       Delete(ctx context.Context, communityID uuid.UUID, agentID uuid.UUID) error
       ListByCommunity(ctx context.Context, communityID uuid.UUID) ([]*model.CommunityAssignment, error)
       CountHubs(ctx context.Context, communityID uuid.UUID) (int, error)
       CountByCommunity(ctx context.Context, communityID uuid.UUID) (int, error)
   }
   ```

7. **DB migration**: Create the `community_assignments` table:
   ```sql
   CREATE TABLE community_assignments (
       community_id  UUID          NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
       agent_id      UUID          NOT NULL REFERENCES agents(id)      ON DELETE CASCADE,
       tenant_id     VARCHAR(255)  NOT NULL,
       role          TEXT          NOT NULL,
       informed_at   TIMESTAMPTZ,
       assigned_at   TIMESTAMPTZ   NOT NULL,
       PRIMARY KEY (community_id, agent_id)
   );
   CREATE INDEX idx_community_assignments_agent_id  ON community_assignments(agent_id);
   CREATE INDEX idx_community_assignments_tenant_id ON community_assignments(tenant_id);
   ```
   Temporarily retain `agents.role` as a deprecated column (populated by migration trigger for backward compat); schedule removal in M6.5 cleanup.

8. **Operator**: The `TacitoAgent` CRD `spec.role` field is populated by Keeper from the assignment record at reconciliation time. The Operator sets `TS_AGENT_ROLE` env var from `spec.role`. Update the kubebuilder validation marker to include `standalone` and remove the static default:
   ```go
   // +kubebuilder:validation:Enum=hub;spoke;standalone
   Role string `json:"role,omitempty"`
   ```

## Acceptance Criteria

1. Creating an agent via POST /api/v1/agents does not accept or return a `role` field.
2. Assigning an agent to a `single-agent` community automatically sets role = `standalone` on the assignment record, regardless of the role value supplied by the caller.
3. Assigning an agent to a `hub-spoke` community requires an explicit `role` field (`hub` or `spoke`).
4. A second `hub` assignment to an existing hub-spoke community is rejected with `409 Conflict`.
5. Assigning a `hub` role to a `single-agent` community is rejected with `409 Conflict`.
6. Unassigning an agent (`DELETE /api/v1/communities/{id}/agents/{agent_id}`) removes the assignment record and returns `204 No Content`.
7. `GET /api/v1/communities/{id}/agents` returns the full assignment list including `role`, `assigned_at`, and `informed_at` per entry.
8. The Operator sets `TS_AGENT_ROLE` from the assignment's role value on pod creation.
9. The `community_assignments` table carries `tenant_id` on every row; all repository queries filter by tenant.

## Test Plan

- **Unit**: `CommunityAssignment.Validate()` enforces topology/role rules.
- **Unit**: Assignment service rejects duplicate hub with domain error → `409`.
- **Unit**: Assignment service rejects `hub` role for `single-agent` community → `409`.
- **Unit**: Assignment service auto-sets `standalone` for `single-agent` topology regardless of caller input.
- **Unit**: `AssignmentUseCase.ListByCommunity` returns structured assignment list.
- **Integration**: Keeper API end-to-end for assignment, unassignment, and list.
- **Integration**: `community_assignments` rows carry correct `tenant_id`.
- **Integration**: Operator reconciler reads role from CRD spec and sets `TS_AGENT_ROLE`.

## API Contract

```
POST   /api/v1/communities/{id}/agents
       Body: { "agent_id": "uuid", "role": "hub|spoke" }  // role omitted for single-agent topology
       Response 201: { "agent_id": "uuid", "role": "...", "assigned_at": "..." }
       Response 409: { "error": "a hub agent is already assigned to this community" }
       Response 409: { "error": "hub role is not valid for single-agent topology" }

DELETE /api/v1/communities/{id}/agents/{agent_id}
       Response 204

GET    /api/v1/communities/{id}/agents
       Response 200: [{ "agent_id": "uuid", "role": "...", "assigned_at": "...", "informed_at": "..." }]
```

## Files Affected

- `internal/keeper/domain/model/agent.go` [MODIFY] — remove `Role` field and its validation
- `internal/keeper/domain/model/community_assignment.go` [NEW] — `AgentRole` type + `CommunityAssignment` aggregate with `Validate()`
- `internal/keeper/application/ports/inbound/usecases.go` [MODIFY] — update `AssignmentUseCase` with `role` parameter on `Assign` and new `ListByCommunity` method
- `internal/keeper/application/ports/outbound/community_assignment_repository.go` [NEW] — `CommunityAssignmentRepository` outbound port
- `internal/keeper/application/service/community_service.go` [MODIFY] — assignment validation logic (topology/role enforcement, hub count check)
- `internal/keeper/adapters/inbound/http/assignment_handlers.go` [MODIFY] — bind role from request body; return `201 Created`; add `ListAssignments` handler
- `internal/keeper/adapters/outbound/postgres/` [NEW] — `community_assignment_repository.go` pgx implementation
- `pkg/kubernetes/apis/tacito/v1alpha1/tacitoagent_types.go` [MODIFY] — update `spec.role` kubebuilder enum to `hub;spoke;standalone`; remove static default
- DB migration [NEW] — `community_assignments` table with `tenant_id`, `informed_at`, indexes
