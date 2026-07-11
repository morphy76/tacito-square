# TASK-M6.5.4.2: Domain Service — Skill Resolution (`skill_resolution.go`)

| Field       | Value |
|-------------|-------|
| ID          | TASK-M6.5.4.2 |
| Status      | TODO |
| Spec        | SPEC-FR-M6.5.4 |
| Depends On  | TASK-M6.5.4.1 |

## Description

Implement the `ResolveAgentSkills` domain service in `internal/keeper/domain/service/skill_resolution.go`. This service encapsulates the union-without-duplication logic specified in §6 and is the single source of truth for effective skill list assembly. It follows the exact same pattern as `prompt_resolution.go`.

## Work Items

### 1. Local Port Interface

Define a local `SkillRepository` interface inside the `service` package (prevents domain from importing application layer):

```go
// SkillRepository defines the outbound port interface required by the skill resolution domain service.
type SkillRepository interface {
    GetByID(ctx context.Context, id uuid.UUID) (*model.Skill, error)
    ResolveCollectionSkills(ctx context.Context, collectionID uuid.UUID) ([]*model.Skill, error)
}
```

### 2. `ResolvedSkill` Value Object

Add `ResolvedSkill` to `internal/keeper/domain/model/skill.go`:

```go
// ResolvedSkill represents a skill in an agent's effective skill list, enriched with source metadata.
type ResolvedSkill struct {
    ID           uuid.UUID   `json:"id"`
    Name         string      `json:"name"`
    Description  string      `json:"description"`
    Content      string      `json:"content"`
    Status       SkillStatus `json:"status"`
    Source       string      `json:"source"` // "collection" or "individual"
    CollectionID *uuid.UUID  `json:"collection_id"`
}
```

### 3. `ResolveAgentSkills` Function

```go
// ResolveAgentSkills resolves the effective skill list for an agent, preserving order and deduplicating.
func ResolveAgentSkills(ctx context.Context, agent *model.Agent, skillRepo SkillRepository) ([]*model.ResolvedSkill, error)
```

Resolution algorithm (strict §6 order):

1. Iterate `agent.SkillCollections` in slice order (attachment position).
2. For each collection call `skillRepo.ResolveCollectionSkills(ctx, colID)` — this already filters for `active` status.
3. For each returned skill, if not already seen (by skill ID), add to result with `Source="collection"`, `CollectionID=&colID`.
4. If skill status is `suspended` or `inactive`, emit `log.Warn().Msgf("skipping non-active skill %s in agent resolution", skill.ID)` and skip.
5. Iterate `agent.Skills` in slice order.
6. For each skill ID, if not already seen, call `skillRepo.GetByID`. If skill is not `active`, log WARN and skip. Otherwise add with `Source="individual"`, `CollectionID=nil`.

### 4. Unit Tests

Create `internal/keeper/domain/service/skill_resolution_test.go`:

- `TestResolveSkills_UnionWithoutDuplication` — collection + individual with same skill ID; appears once under `source="collection"`.
- `TestResolveSkills_CollectionOrder` — two collections; skills from first collection appear before second.
- `TestResolveSkills_SkipsSuspended` — mock returns suspended skill; absent from result, no error, WARN emitted.
- `TestResolveSkills_SkipsInactive` — mock returns inactive skill; absent from result, no error, WARN emitted.
- `TestResolveSkills_MultipleCollections` — two collections sharing a skill; skill appears only once.
- `TestResolveSkills_EmptyAgent` — agent with no skills or collections; returns empty slice.

Use a mock `SkillRepository` implemented inline (struct with function fields, similar to prompt resolution tests).

## Acceptance Criteria

1. All 6 unit tests pass in RED→GREEN→REFACTOR order.
2. Domain service package has zero imports from `application` or `adapters` layers.
3. `ResolvedSkill` struct is in `domain/model/skill.go`.
