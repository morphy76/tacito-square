# SPEC-FR-M6.5.11: Community Assignment Transaction + Stale Marker

| Field       | Value |
|-------------|-------|
| ID          | SPEC-FR-M6.5.11 |
| Status      | DRAFT |
| Milestone   | M6.5 |
| Component   | keeper, operator |
| Depends On  | SPEC-FR-M6.5.1, SPEC-FR-M6.5.9, SPEC-FR-M6.5.10 |
| Supersedes  | none |

## Context

Assigning an agent to a community is a multi-step distributed operation spanning Keeper (DB write + NATS event) and the Operator (CRD reconcile + pod materialization). Any step can fail. Additionally, when supporting entities (skills, prompts, MCP clients, LLM bindings) are updated in Keeper after an agent has already been deployed, the running agent has a stale environment. This spec defines the full assignment transaction semantics and the lightweight stale-marker mechanism.

## Specification

### 1. Assignment Transaction

The Keeper `POST /api/v1/communities/{id}/agents` operation executes atomically within a DB transaction:

1. **Validate topology constraints**: single-agent = max 1 agent; hub-spoke = max 1 hub, role must be `hub` or `spoke`.
2. **Validate LLM binding**: the agent's `Brain.LLMBindingID` must reference an LLM binding with `status = active`. Reject with `422` if inactive.
3. **Validate supporting entities**: all attached skills, prompts, and MCP clients must exist and have `status = active`. Inactive ones produce warnings, not errors (they will be excluded from resolution).
4. **Determine effective tier** (per SPEC-FR-M6.5.10).
5. **Write `community_assignments` record**: `{community_id, agent_id, role, effective_tier, informed_at = NOW(), assigned_at = NOW()}`.
6. **Update `agents.status = assigned`**.
7. **Commit DB transaction**.
8. **Emit NATS event** `ts.keeper.agent.assigned` with payload `{agent_id, community_id, role, tenant_id}`.
9. **Update `TacitoAgent` CRD** `spec.role` and `spec.communityID` to trigger Operator reconciliation.

If steps 1–7 fail, the DB transaction is rolled back. The agent remains in `defined` status.

### 2. Stale Marker

When any supporting entity changes in Keeper (LLM binding updated/deactivated, skill/prompt/MCP client content changed or deactivated):

1. Keeper identifies all `community_assignments` referencing the changed entity.
2. Sets `community_assignments.informed_at = NULL` for each affected assignment.
3. Sets `agents.status = stale` for each affected agent.

Rules:
- A `stale` agent **continues to function normally** — no forced shutdown or request rejection.
- The stale marker is **informational only**, surfaced in API responses and the eventual UI.
- To clear stale status, the user triggers a **redeploy** via `POST /api/v1/communities/{id}/agents/{agent_id}/redeploy`. This causes the Operator to re-fetch all supporting entity state from Keeper, rebuild the environment ConfigMaps, and roll the pod. On successful reconciliation, Keeper sets `informed_at = NOW()` and `agents.status = assigned`.

### 3. Unassignment

`DELETE /api/v1/communities/{id}/agents/{agent_id}`:
1. Removes the `community_assignments` record.
2. Sets `agents.status = defined` and clears `agents.community_id`.
3. Emits `ts.keeper.agent.unassigned` NATS event.
4. The Operator deletes the `TacitoAgent` pod (or scales to 0, depending on operator reconciler policy).

## Acceptance Criteria

1. Assignment with an inactive LLM binding returns `422` with `{"error": "llm binding is not active"}`.
2. Successful assignment creates a `community_assignments` record with `informed_at` set.
3. Updating a skill attached to an assigned agent sets `agents.status = stale` and `informed_at = NULL`.
4. `GET /api/v1/agents/{id}` returns `"status": "stale"` for the affected agent.
5. `POST /api/v1/communities/{id}/agents/{agent_id}/redeploy` clears the stale status.
6. A stale agent continues to respond to NATS messages and process requests.
7. DB transaction rollback on step 2 failure leaves agent in `defined` status.

## Test Plan

- **Unit**: Assignment service rejects inactive LLM binding with domain error.
- **Unit**: Stale marker propagation logic for each entity type (skill, prompt, MCP client, LLM binding).
- **Unit**: Transaction rollback simulation.
- **Integration**: Full assignment API flow end-to-end.
- **Integration**: Entity update → stale marker → redeploy → stale cleared.

## API Contract

```
POST   /api/v1/communities/{id}/agents
       Body: { agent_id, role, tier? }
       Response 200: { agent_id, role, effective_tier, assigned_at, warnings[] }
       Response 409: { error: "a hub agent is already assigned..." }
       Response 422: { error: "llm binding is not active" }

DELETE /api/v1/communities/{id}/agents/{agent_id}   → 204

POST   /api/v1/communities/{id}/agents/{agent_id}/redeploy
       Response 202: { message: "redeploy triggered" }

GET    /api/v1/agents/{id}
       Response 200: { ..., status: "assigned|stale|defined", informed_at: "..." }
```

## Files Affected

- `internal/keeper/application/service/community_service.go` [MODIFY] — full transaction + stale logic
- `internal/keeper/domain/model/community_assignment.go` [MODIFY] — `InformedAt *time.Time`
- `internal/keeper/domain/model/agent.go` [MODIFY] — add `stale` to agent status enum
- `internal/keeper/adapters/inbound/http/community_handler.go` [MODIFY] — redeploy endpoint
- DB migration [MODIFY] — add `informed_at` to `community_assignments`; add `stale` to `agents.status` enum
