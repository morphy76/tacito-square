# TASK-M6.5.4.1: DB Migration — `agent_skill_collections` Table + Agent Model Field

| Field       | Value |
|-------------|-------|
| ID          | TASK-M6.5.4.1 |
| Status      | IMPLEMENTED |
| Spec        | SPEC-FR-M6.5.4 |
| Depends On  | none |

## Description

Create a goose migration to introduce the `agent_skill_collections` join table and update the `Agent` domain model to carry `SkillCollections []uuid.UUID`. The skills tables (`skills`, `skill_collections`, `skill_collection_skills`, `agent_skills`) already exist in `00001_init.sql`.

## Work Items

### 1. Migration File

Create `deploy/postgres/migrations/00002_agent_skill_collections.sql`.

**Up block:**

```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS agent_skill_collections (
    agent_id            UUID    NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    skill_collection_id UUID    NOT NULL REFERENCES skill_collections(id) ON DELETE CASCADE,
    position            INT     NOT NULL DEFAULT 0,
    PRIMARY KEY (agent_id, skill_collection_id)
);

CREATE INDEX IF NOT EXISTS idx_agent_skill_collections_agent_id
    ON agent_skill_collections(agent_id);
```

**Down block:**

```sql
-- +goose Down
DROP TABLE IF EXISTS agent_skill_collections;
```

### 2. Agent Domain Model

In `internal/keeper/domain/model/agent.go`, add the `SkillCollections` field to the `Agent` struct after the existing `Skills` field:

```go
Skills           []uuid.UUID `json:"skills,omitempty"`
SkillCollections []uuid.UUID `json:"skill_collections,omitempty"`
```

No changes to `Validate()` are required — an empty `SkillCollections` slice is valid.

### 3. Agent Repository Postgres Adapter

In `internal/keeper/adapters/outbound/postgres/agent_repository.go`, ensure that `SkillCollections` is populated when loading an agent (analogous to how `Skills` is loaded). Add a helper `loadAgentSkillCollections(ctx, agentID)` that queries:

```sql
SELECT skill_collection_id
FROM agent_skill_collections
WHERE agent_id = $1
ORDER BY position ASC
```

Call this helper in `GetByID`, `GetByName`, and `List` (after loading `Skills`).

## Acceptance Criteria

1. Migration runs `Up` and `Down` without errors via `goose`.
2. After running `Up`, `agent_skill_collections` table exists with `position` column and correct FK constraints.
3. `Agent.SkillCollections` is populated from the DB in `GetByID`, `GetByName`, and `List`.
4. Existing agent integration tests remain green.
