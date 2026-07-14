# TASK-M6.5.4.3: Ports & Application Service — Collection-Agent Ops + PATCH Status

| Field       | Value |
|-------------|-------|
| ID          | TASK-M6.5.4.3 |
| Status      | IMPLEMENTED |
| Spec        | SPEC-FR-M6.5.4 |
| Depends On  | TASK-M6.5.4.1, TASK-M6.5.4.2 |

## Description

Extend the `SkillUseCase` inbound port, `SkillRepository` outbound port, and `SkillService` application service to support: (a) collection-to-agent attachment/detachment, (b) resolved skill list retrieval via the domain resolution service, and (c) skill status transition (PATCH). Also extend the postgres adapter with the two new collection-agent association queries.

## Work Items

### 1. Extend `SkillUseCase` Inbound Port

In `internal/keeper/application/ports/inbound/usecases.go`, add to `SkillUseCase`:

```go
// Agent–SkillCollection associations
AttachCollectionToAgent(ctx context.Context, agentID uuid.UUID, collectionID uuid.UUID) error
DetachCollectionFromAgent(ctx context.Context, agentID uuid.UUID, collectionID uuid.UUID) error

// Resolved effective skill list
ResolveAgentSkills(ctx context.Context, agentID uuid.UUID) ([]*model.ResolvedSkill, error)

// Status transition
PatchStatus(ctx context.Context, id uuid.UUID, status model.SkillStatus) (*model.Skill, error)
```

### 2. Extend `SkillRepository` Outbound Port

In `internal/keeper/application/ports/outbound/repositories.go`, add to `SkillRepository`:

```go
// Agent-SkillCollection associations
AttachCollectionToAgent(ctx context.Context, agentID uuid.UUID, collectionID uuid.UUID) error
DetachCollectionFromAgent(ctx context.Context, agentID uuid.UUID, collectionID uuid.UUID) error

// Collection membership management
AddSkillToCollection(ctx context.Context, collectionID uuid.UUID, skillID uuid.UUID) error
RemoveSkillFromCollection(ctx context.Context, collectionID uuid.UUID, skillID uuid.UUID) error
```

### 3. Extend `SkillService` Application Service

In `internal/keeper/application/service/skill_service.go`:

- Inject `AgentRepository` into `SkillService` constructor (needed to fetch agent for resolution).
- Add `AttachCollectionToAgent`, `DetachCollectionFromAgent` — delegate to `repo`.
- Add `ResolveAgentSkills`:
  1. Fetch agent via `agentRepo.GetByID`.
  2. Create a `skillRepoAdapter` (adapting `SkillRepository` to the domain service's local port interface).
  3. Call `domainsrv.ResolveAgentSkills(ctx, agent, adapter)`.
- Add `PatchStatus`:
  1. Fetch skill by ID.
  2. Validate new status.
  3. Set `skill.Status = newStatus` and `skill.UpdatedAt = time.Now().UTC()`.
  4. Call `repo.Update(ctx, skill)`.
  5. Return updated skill.
- Add `AddSkillToCollection`, `RemoveSkillFromCollection` — delegate to `repo`.

### 4. Extend Postgres Adapter

In `internal/keeper/adapters/outbound/postgres/skill_repository.go`, add:

**`AttachCollectionToAgent`:**
```sql
INSERT INTO agent_skill_collections (agent_id, skill_collection_id, position)
SELECT $1, $2,
    COALESCE((SELECT MAX(position) + 1 FROM agent_skill_collections WHERE agent_id = $1), 0)
WHERE EXISTS (SELECT 1 FROM skill_collections WHERE id = $2 AND tenant_id = $3)
ON CONFLICT (agent_id, skill_collection_id) DO NOTHING
```
Return `not found` error if `RowsAffected() == 0`.

**`DetachCollectionFromAgent`:**
```sql
DELETE FROM agent_skill_collections
WHERE agent_id = $1 AND skill_collection_id = $2
  AND EXISTS (SELECT 1 FROM skill_collections WHERE id = $2 AND tenant_id = $3)
```

**`AddSkillToCollection`:**
```sql
INSERT INTO skill_collection_skills (skill_collection_id, skill_id) VALUES ($1, $2)
ON CONFLICT (skill_collection_id, skill_id) DO NOTHING
```
Return `HTTP 409`-triggerable error (e.g. `"skill already member of collection: %s"`) if `RowsAffected() == 0`.

**`RemoveSkillFromCollection`:**
```sql
DELETE FROM skill_collection_skills WHERE skill_collection_id = $1 AND skill_id = $2
```

## Acceptance Criteria

1. `SkillUseCase` and `SkillRepository` interfaces compile cleanly with all new methods.
2. `SkillService.ResolveAgentSkills` correctly delegates to the domain resolution service.
3. `SkillService.PatchStatus` returns `not found` error for unknown IDs and `invalid status` for bad values.
4. Postgres adapter methods have correct tenant-scoping guards.
5. Existing `skill_service.go` logic is not broken — all previously passing service tests remain green.
