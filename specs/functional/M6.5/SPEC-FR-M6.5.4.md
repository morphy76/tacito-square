# SPEC-FR-M6.5.4: Skill & SkillCollection CRUD APIs + Agent Association

| Field       | Value                                           |
|-------------|-------------------------------------------------|
| ID          | SPEC-FR-M6.5.4                                  |
| Status      | DRAFT                                           |
| Milestone   | M6.5                                            |
| Component   | keeper                                          |
| Depends On  | SPEC-FR-M3.3                                    |
| Supersedes  | SPEC-FR-M9.8 (DRAFT), SPEC-FR-M9.15 (DRAFT)    |

---

## Context

`Skill` and `SkillCollection` domain models already exist in `internal/keeper/domain/model/skill.go`. Skills are **procedural knowledge** injected into the system prompt at runtime when the LLM emits an `enable_skill` tool call. They are semantically distinct from MCP clients: skills affect the system prompt (procedural memory injection); MCP clients execute external actions.

A hub agent may legitimately carry skills. Routing skills help the hub decide which spoke to delegate to (e.g., "if the request is about billing, delegate to the billing-agent"). Attaching skills to a hub is therefore valid and must succeed, but the system emits a `warnings[]` field in the hub-assignment response to surface this fact for operator awareness.

The current `Agent` model carries `Skills []uuid.UUID` (direct skill IDs). This spec adds `SkillCollections []uuid.UUID`, the full CRUD REST APIs, and the agent-association endpoints.

---

## Specification

### 1. Skill CRUD

All endpoints are tenant-scoped. Tenant ID resolved from JWT claims or `X-Tenant-ID` header, propagated via `context.Context`.

| Method | Path                 | Description                         |
|--------|----------------------|-------------------------------------|
| GET    | /api/v1/skills       | List non-inactive skills for tenant |
| POST   | /api/v1/skills       | Create a new skill                  |
| GET    | /api/v1/skills/{id}  | Get a single skill by ID            |
| PUT    | /api/v1/skills/{id}  | Update skill (metadata or status)   |
| DELETE | /api/v1/skills/{id}  | Soft-delete (set status=inactive)   |

**Status lifecycle:** `active` / `suspended` / `inactive`. A `suspended` skill is still persisted but is excluded from agent resolution (treated as non-active). An `inactive` skill does not appear in list responses.

### 2. SkillCollection CRUD

| Method | Path                                                          | Description                          |
|--------|---------------------------------------------------------------|--------------------------------------|
| GET    | /api/v1/skill-collections                                     | List skill collections for tenant    |
| POST   | /api/v1/skill-collections                                     | Create a new skill collection        |
| GET    | /api/v1/skill-collections/{id}                               | Get a single collection by ID        |
| PUT    | /api/v1/skill-collections/{id}                               | Update collection metadata           |
| DELETE | /api/v1/skill-collections/{id}                               | Soft-delete collection               |
| POST   | /api/v1/skill-collections/{id}/skills/{skill_id}             | Add a skill to the collection        |
| DELETE | /api/v1/skill-collections/{id}/skills/{skill_id}             | Remove a skill from the collection   |

Membership management endpoints return HTTP 200 with the updated collection object. Adding a skill that is already a member returns HTTP 409.

### 3. Agent Association Endpoints

| Method | Path                                                              | Description                                      |
|--------|-------------------------------------------------------------------|--------------------------------------------------|
| POST   | /api/v1/agents/{id}/skills/{skill_id}                             | Attach individual skill to agent                 |
| DELETE | /api/v1/agents/{id}/skills/{skill_id}                             | Detach individual skill from agent               |
| POST   | /api/v1/agents/{id}/skill-collections/{collection_id}             | Attach collection to agent                       |
| DELETE | /api/v1/agents/{id}/skill-collections/{collection_id}             | Detach collection from agent                     |
| GET    | /api/v1/agents/{id}/skills                                        | Returns the effective resolved skill list        |

`GET /api/v1/agents/{id}/skills` returns each resolved skill with: `id`, `name`, `description`, `content`, `status`, `source` (`"collection"` or `"individual"`), `collection_id` (if source is collection).

### 4. Agent Model Change

Add `SkillCollections []uuid.UUID` to the `Agent` aggregate alongside the existing `Skills []uuid.UUID`:

```go
Skills           []uuid.UUID `json:"skills,omitempty"`
SkillCollections []uuid.UUID `json:"skill_collections,omitempty"`
```

No existing field is removed. A DB migration must add the `agent_skill_collections` join table.

### 5. Resolution Semantics at Agent Spawn

When Keeper assembles the `PropagatedAgentConfig`, it resolves the effective skill list as follows:

1. For each entry in `SkillCollections` (in agent attachment order), iterate the collection's ordered member list. Include each `active` skill not yet in the result set (deduplication by skill ID).
2. For each directly attached skill in `Skills` (in attachment order), include it only if not already in the result set.
3. Skills with status `suspended` or `inactive` are **silently skipped** with a `WARN` log entry: `"skipping non-active skill {id} in agent resolution"`.
4. The ordered result is delivered in `PropagatedAgentConfig.Skills` as `[]{ID, Name, Description, Content}`.

### 6. Hub Assignment Warning

When an agent with skills attached is assigned the hub role (via the existing hub-assignment endpoint), the assignment **succeeds** (HTTP 200). The response body includes a `warnings` array:

```json
{
  "agent_id": "agent-uuid",
  "role": "hub",
  "warnings": [
    "agent has skills attached; skills are valid for hub routing logic and will be injected into the hub system prompt"
  ]
}
```

This is not an error. The assignment proceeds and the skills are included in the hub's `PropagatedAgentConfig`.

### 7. Caching

- Cache resolved skill list per agent in Redis: key `agent-skills:{tenantID}:{agentID}`, TTL `60s` (Viper key: `cache.agent_skills_ttl`).
- Invalidate on: any agent-skill or agent-skill-collection attachment/detachment; any skill status change; any collection membership change.

---

## Acceptance Criteria

1. **Skill lifecycle**: `POST /api/v1/skills` creates a skill. `PUT /api/v1/skills/{id}` with `{"status": "suspended"}` transitions to `suspended`. Suspended skill absent from resolved list.
2. **Agent association**: Attaching a skill to an agent; `GET /api/v1/agents/{id}/skills` returns it with full `name`, `description`, `content` fields.
3. **Union-without-duplication**: Same skill attached individually and via collection appears only once in resolved list, listed under `source="collection"`.
4. **Hub warning**: Assigning an agent with skills as hub returns HTTP 200 with a non-empty `warnings[]` field.
5. **Active-only filter**: Suspended skill absent from resolved list; no error returned.
6. **Skill content in resolved list**: Each item in `GET /api/v1/agents/{id}/skills` includes the `content` field (required for `PropagatedAgentConfig` assembly).
7. **Cache invalidation**: After detaching a skill collection, subsequent resolved list reflects the removal.

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
  - `TestAssignHubRole_WithSkills_Returns200WithWarning` — mock returns agent with skills; response contains `warnings[]`.
  - `TestAssignHubRole_NoSkills_Returns200NoWarning` — no skills; `warnings` empty or absent.

### Integration Tests (Gin TestMode + httptest)

- `internal/keeper/adapters/inbound/http/skill_handler_test.go`
  - `TestCreateSkill_Success` — verifies 201 and response shape including `content`.
  - `TestSuspendSkill_ExcludedFromResolution` — attach, suspend, verify absent in resolved list.
  - `TestAddSkillToCollection_Duplicate_Returns409`.
  - `TestAttachSkillToAgent_GetResolvedList` — full E2E via handler.
  - `TestHubAssignment_WithSkills_WarningPresent` — verifies `warnings` field in response.
  - `TestCacheInvalidationOnSkillDetach` — mock Redis; DEL called on detach.

---

## API Contract

### POST /api/v1/skills

**Request Body:**
```json
{
  "name": "billing-routing",
  "description": "Routes billing-related queries to the billing-agent spoke",
  "content": "When the user's request relates to invoices, payments, subscriptions, or billing disputes, delegate to the billing-agent using the delegate_to_agent tool."
}
```

**Response 201:**
```json
{
  "id": "772a0600-e29b-41d4-a716-446655440002",
  "tenant_id": "tenant-abc",
  "name": "billing-routing",
  "description": "Routes billing-related queries to the billing-agent spoke",
  "content": "When the user's request relates to invoices...",
  "status": "active",
  "created_at": "2026-07-10T08:00:00Z",
  "updated_at": "2026-07-10T08:00:00Z"
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

### Hub Assignment Response with Warning

**POST /api/v1/agents/{id}/role — Response 200:**
```json
{
  "agent_id": "agent-uuid",
  "role": "hub",
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
| `internal/keeper/domain/model/skill.go`                                   | EXISTING — no structural changes                                    |
| `internal/keeper/domain/model/agent.go`                                   | MODIFY — add `SkillCollections []uuid.UUID`                         |
| `internal/keeper/domain/service/skill_resolution.go`                      | NEW — union-without-duplication resolution logic                    |
| `internal/keeper/application/ports/inbound/skill_service.go`              | NEW — inbound port interface                                        |
| `internal/keeper/application/ports/outbound/skill_repository.go`          | NEW — outbound port interface                                       |
| `internal/keeper/application/service/skill_service.go`                    | NEW — use case orchestration                                        |
| `internal/keeper/adapters/inbound/http/skill_handler.go`                  | NEW — Gin HTTP handler                                              |
| `internal/keeper/adapters/outbound/postgres/skill_repository.go`          | NEW — pgx adapter                                                   |
| `internal/keeper/adapters/outbound/redis/skill_cache.go`                  | NEW — Redis cache adapter                                           |
| `db/migrations/<timestamp>_create_skills.sql`                             | NEW — skills, skill_collections, skill_collection_members, agent_skills, agent_skill_collections tables |
