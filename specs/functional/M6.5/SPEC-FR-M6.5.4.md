# SPEC-FR-M6.5.4: Skill & SkillCollection CRUD APIs + Agent Association

| Field       | Value                                           |
|-------------|-------------------------------------------------|
| ID          | SPEC-FR-M6.5.4                                  |
| Status      | ACCEPTED                                        |
| Milestone   | M6.5                                            |
| Component   | keeper                                          |
| Depends On  | SPEC-FR-M3.3                                    |
| Supersedes  | SPEC-FR-M9.8 (DRAFT), SPEC-FR-M9.15 (DRAFT)    |

---

## Context

`Skill` and `SkillCollection` domain models already exist in `internal/keeper/domain/model/skill.go`. Skills are **procedural knowledge** injected into the system prompt at runtime when the LLM emits an `enable_skill` tool call. They are semantically distinct from MCP clients: skills affect the system prompt (procedural memory injection); MCP clients execute external actions.

A hub agent may legitimately carry skills. Routing skills help the hub decide which spoke to delegate to (e.g., "if the request is about billing, delegate to the billing-agent"). Attaching skills to a hub is therefore valid and must succeed, but the system emits a `warnings[]` field in the hub-assignment response to surface this fact for operator awareness.

The current `Agent` model carries `Skills []uuid.UUID` (direct skill IDs). This spec adds `SkillCollections []uuid.UUID`, the missing association endpoints for skill collections on agents, the union-resolution domain service, the PATCH status endpoint, and the collection-aware skill assembly in the CRD coordinator. Core Skill and SkillCollection CRUD (handlers, service, ports, postgres adapter) are already implemented and are **not repeated** here.

---

## Specification

### 1. Skill CRUD — Existing (no changes)

All five CRUD endpoints are already implemented:

| Method | Path                 | Description                         |
|--------|----------------------|-------------------------------------|
| GET    | /api/v1/skills       | List non-inactive skills for tenant |
| POST   | /api/v1/skills       | Create a new skill                  |
| GET    | /api/v1/skills/{id}  | Get a single skill by ID            |
| PUT    | /api/v1/skills/{id}  | Update skill (metadata only)        |
| DELETE | /api/v1/skills/{id}  | Soft-delete (set status=inactive)   |

### 2. Skill Status Transition — New

Status transitions are a partial mutation and must use a dedicated endpoint:

| Method | Path                        | Description                         |
|--------|-----------------------------|-------------------------------------|
| PATCH  | /api/v1/skills/{id}         | Transition skill status             |

**Status lifecycle:** `active` / `suspended` / `inactive`. A `suspended` skill is still persisted but is excluded from agent resolution. An `inactive` skill does not appear in list responses. The `PATCH` body accepts only `{"status": "<value>"}`.

### 3. SkillCollection CRUD — Existing + Membership Endpoints (new)

The five CRUD endpoints are already implemented. The following collection-membership management endpoints are new:

| Method | Path                                                          | Description                          |
|--------|---------------------------------------------------------------|--------------------------------------|
| POST   | /api/v1/skill-collections/{id}/skills/{skill_id}             | Add a skill to the collection        |
| DELETE | /api/v1/skill-collections/{id}/skills/{skill_id}             | Remove a skill from the collection   |

Membership management endpoints return HTTP 200 with the updated collection object. Adding a skill that is already a member returns HTTP 409.

### 4. Agent Association Endpoints — Partial New

Individual skill attach/detach on agents are already implemented. The following are new:

| Method | Path                                                              | Description                                      |
|--------|-------------------------------------------------------------------|--------------------------------------------------|
| POST   | /api/v1/agents/{id}/skill-collections/{collection_id}             | Attach collection to agent                       |
| DELETE | /api/v1/agents/{id}/skill-collections/{collection_id}             | Detach collection from agent                     |
| GET    | /api/v1/agents/{id}/skills                                        | Returns the effective resolved skill list        |

`GET /api/v1/agents/{id}/skills` returns each resolved skill with: `id`, `name`, `description`, `content`, `status`, `source` (`"collection"` or `"individual"`), `collection_id` (if source is collection).

### 5. Agent Model Change

Add `SkillCollections []uuid.UUID` to the `Agent` aggregate alongside the existing `Skills []uuid.UUID`:

```go
Skills           []uuid.UUID `json:"skills,omitempty"`
SkillCollections []uuid.UUID `json:"skill_collections,omitempty"`
```

No existing field is removed. A DB migration must add the `agent_skill_collections` join table with a `position INT` column to preserve attachment order.

### 6. Resolution Semantics at Agent Spawn

When Keeper assembles the resolved skill list (via `GET /api/v1/agents/{id}/skills` and in the CRD coordinator), it resolves the effective skill list via the domain service `skill_resolution.go` as follows:

1. For each entry in `SkillCollections` (in agent attachment order, ascending `position`), iterate the collection's ordered member list. Include each `active` skill not yet in the result set (deduplication by skill ID).
2. For each directly attached skill in `Skills` (in attachment order), include it only if not already in the result set.
3. Skills with status `suspended` or `inactive` are **silently skipped** with a `WARN` log entry: `"skipping non-active skill {id} in agent resolution"`.
4. The ordered result is delivered in `PropagatedAgentConfig.Skills` as `[]{ID, Name, Description, Content}`.

The domain service defines its own local `SkillRepository` interface (following the established pattern of `prompt_resolution.go`) to avoid domain importing application-layer packages.

### 7. Hub Assignment Warning

When an agent with skills attached (individual or via collection) is assigned the hub role (via `POST /api/v1/communities/{community_id}/agents` with `"role": "hub"`), the assignment **succeeds** (HTTP 201). The response body includes a `warnings` array:

```json
{
  "agent_id": "agent-uuid",
  "role": "hub",
  "assigned_at": "2026-07-11T08:00:00Z",
  "warnings": [
    "agent has skills attached; skills are valid for hub routing logic and will be injected into the hub system prompt"
  ]
}
```

This is not an error. The assignment proceeds and the skills are included in the hub's `PropagatedAgentConfig`. The warning is only emitted if `len(agent.Skills) > 0 || len(agent.SkillCollections) > 0`.

### 8. CRD Coordinator — Collection-Aware Skill Resolution

`ResolveAndSynthesizeSystemPrompt` in `crd_coordinator.go` currently iterates `agent.Skills` directly. It must be updated to call the new `skill_resolution.go` domain service so that skills attached via `SkillCollections` are also included.

---

## Acceptance Criteria

1. **Skill status transition**: `PATCH /api/v1/skills/{id}` with `{"status": "suspended"}` transitions to `suspended`. Suspended skill absent from resolved list.
2. **Agent–collection association**: Attaching a collection to an agent; `GET /api/v1/agents/{id}/skills` returns all active skills from that collection.
3. **Union-without-duplication**: Same skill attached individually and via collection appears only once in resolved list, listed under `source="collection"`.
4. **Hub warning**: Assigning an agent with skills as hub returns HTTP 201 with a non-empty `warnings[]` field.
5. **Active-only filter**: Suspended skill absent from resolved list; no error returned.
6. **Skill content in resolved list**: Each item in `GET /api/v1/agents/{id}/skills` includes the `content` field (required for `PropagatedAgentConfig` assembly).
7. **Collection membership add/remove**: `POST/DELETE /api/v1/skill-collections/{id}/skills/{skill_id}` return HTTP 200 with updated collection. Duplicate add returns HTTP 409.
8. **CRD coordinator**: `ResolveAndSynthesizeSystemPrompt` uses the domain resolution service and includes skills from both `Skills` and `SkillCollections`.

---

## Test Plan

### Unit Tests

- `internal/keeper/domain/service/skill_resolution_test.go`
  - `TestResolveSkills_UnionWithoutDuplication` — collection + individual with overlap, deduplicated.
  - `TestResolveSkills_CollectionOrder` — collection skills precede individual skills.
  - `TestResolveSkills_SkipsSuspended` — suspended skill absent, no error.
  - `TestResolveSkills_SkipsInactive` — inactive skill absent, WARN logged.
  - `TestResolveSkills_MultipleCollections` — two collections with shared member; appears once.
- `internal/keeper/domain/model/skill_test.go`
  - `TestSkillValidate_StatusValues` — active/suspended/inactive pass; unknown fails.

### Unit Tests — Hub Warning Logic

- `internal/keeper/application/service/agent_service_test.go`
  - `TestAssignHubRole_WithSkills_Returns201WithWarning` — mock returns agent with skills; response contains `warnings[]`.
  - `TestAssignHubRole_NoSkills_Returns201NoWarning` — no skills; `warnings` empty or absent.

### Integration Tests (Gin TestMode + httptest)

- `internal/keeper/adapters/inbound/http/skill_handlers_test.go`
  - `TestPatchSkillStatus_Suspend_ExcludedFromResolution` — PATCH to suspend, verify absent in resolved list.
  - `TestAddSkillToCollection_Duplicate_Returns409`.
  - `TestRemoveSkillFromCollection_Success`.
  - `TestAttachCollectionToAgent_GetResolvedList` — full flow via handler.
  - `TestHubAssignment_WithSkills_WarningPresent` — verifies `warnings` field in assignment response.

---

## API Contract

### PATCH /api/v1/skills/{id}

**Request Body:**
```json
{ "status": "suspended" }
```

**Response 200:**
```json
{
  "id": "skill-uuid",
  "tenant_id": "tenant-abc",
  "name": "billing-routing",
  "status": "suspended",
  "updated_at": "2026-07-11T08:00:00Z"
}
```

### GET /api/v1/agents/{id}/skills

**Response 200:**
```json
{
  "agent_id": "agent-uuid",
  "resolved_skills": [
    {
      "id": "skill-uuid-1",
      "name": "billing-routing",
      "description": "Routes billing queries",
      "content": "When the user's request relates to invoices...",
      "status": "active",
      "source": "collection",
      "collection_id": "collection-uuid-1"
    },
    {
      "id": "skill-uuid-2",
      "name": "tone-calibration",
      "description": "Adjusts response tone",
      "content": "Always respond in a professional and empathetic tone...",
      "status": "active",
      "source": "individual",
      "collection_id": null
    }
  ],
  "total": 2
}
```

### POST /api/v1/communities/{community_id}/agents — Hub Assignment with Warning

**Response 201:**
```json
{
  "agent_id": "agent-uuid",
  "role": "hub",
  "assigned_at": "2026-07-11T08:00:00Z",
  "warnings": [
    "agent has skills attached; skills are valid for hub routing logic and will be injected into the hub system prompt"
  ]
}
```

### Error Response (all endpoints)

```json
{ "error": "descriptive error message" }
```

### OpenAPI Tag

All skill operations carry the tag: `agent-config/skills`
All skill-collection operations carry the tag: `agent-config/skill-collections`

---

## Files Affected

| File                                                                      | Action                                                              |
|---------------------------------------------------------------------------|---------------------------------------------------------------------|
| `internal/keeper/domain/model/agent.go`                                   | MODIFY — add `SkillCollections []uuid.UUID`                         |
| `internal/keeper/domain/model/skill.go`                                   | MODIFY — add `ResolvedSkill` struct                                 |
| `internal/keeper/domain/service/skill_resolution.go`                      | NEW — union-without-duplication resolution domain service           |
| `internal/keeper/domain/service/skill_resolution_test.go`                 | NEW — unit tests for resolution logic                               |
| `internal/keeper/application/ports/inbound/usecases.go`                   | MODIFY — extend `SkillUseCase` with collection-agent ops + PATCH    |
| `internal/keeper/application/ports/outbound/repositories.go`              | MODIFY — extend `SkillRepository` with collection-agent ops         |
| `internal/keeper/application/service/skill_service.go`                    | MODIFY — add collection-agent attach/detach + ResolveAgentSkills    |
| `internal/keeper/application/service/agent_service.go`                    | MODIFY — Assign() emits warnings[] for hub + skills                 |
| `internal/keeper/application/service/agent_service_test.go`               | MODIFY — tests for hub warning                                      |
| `internal/keeper/adapters/inbound/http/skill_handlers.go`                 | MODIFY — add PATCH status, GET resolved list, collection-agent ops  |
| `internal/keeper/adapters/inbound/http/skill_collection_handlers.go`      | MODIFY — add POST/DELETE membership endpoints                       |
| `internal/keeper/adapters/inbound/http/skill_handlers_test.go`            | MODIFY — integration tests for new endpoints                        |
| `internal/keeper/adapters/inbound/http/assignment_handlers.go`            | MODIFY — enrich Assign() response with warnings[]                   |
| `internal/keeper/adapters/outbound/postgres/skill_repository.go`          | MODIFY — add collection-agent association methods                   |
| `internal/keeper/adapters/outbound/crd/crd_coordinator.go`               | MODIFY — use skill_resolution domain service                        |
| `deploy/postgres/migrations/<timestamp>_agent_skill_collections.sql`      | NEW — agent_skill_collections join table with position column       |
