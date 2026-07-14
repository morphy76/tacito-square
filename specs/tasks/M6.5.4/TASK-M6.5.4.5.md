# TASK-M6.5.4.5: CRD Coordinator — Collection-Aware Skill Resolution

| Field       | Value |
|-------------|-------|
| ID          | TASK-M6.5.4.5 |
| Status      | IMPLEMENTED |
| Spec        | SPEC-FR-M6.5.4 |
| Depends On  | TASK-M6.5.4.2 |

## Description

Update `ResolveAndSynthesizeSystemPrompt` in `internal/keeper/adapters/outbound/crd/crd_coordinator.go` to use the new `skill_resolution.go` domain service instead of its current direct `agent.Skills` ID loop. This makes the CRD coordinator aware of `agent.SkillCollections` and gives consistent resolution semantics between the API resolved list and the spawned agent's system prompt.

## Work Items

### 1. Adapter Shim

The domain service `ResolveAgentSkills` requires a `service.SkillRepository` (the local domain-layer interface). The CRD coordinator already holds an `outbound.SkillRepository` (which implements both `GetByID` and `ResolveCollectionSkills`). Create a local adapter struct inside `crd_coordinator.go` that wraps `outbound.SkillRepository` and satisfies the domain service interface:

```go
type skillRepoAdapter struct {
    repo outbound.SkillRepository
}

func (a skillRepoAdapter) GetByID(ctx context.Context, id uuid.UUID) (*model.Skill, error) {
    return a.repo.GetByID(ctx, id)
}

func (a skillRepoAdapter) ResolveCollectionSkills(ctx context.Context, collectionID uuid.UUID) ([]*model.Skill, error) {
    return a.repo.ResolveCollectionSkills(ctx, collectionID)
}
```

### 2. Replace Skill Loop in `ResolveAndSynthesizeSystemPrompt`

Replace the current block (lines ~208-221):

```go
// OLD — direct Skills loop (does not honour SkillCollections)
var skillsList []SkillConfig
if c.skillRepo != nil {
    for _, skillID := range agent.Skills {
        skill, err := c.skillRepo.GetByID(ctx, skillID)
        ...
    }
}
```

With the domain resolution service:

```go
// NEW — collection-aware via domain service
var skillsList []SkillConfig
if c.skillRepo != nil {
    adapter := skillRepoAdapter{repo: c.skillRepo}
    resolved, err := domainsrv.ResolveAgentSkills(ctx, agent, adapter)
    if err != nil {
        return "", fmt.Errorf("resolving agent skills: %w", err)
    }
    for _, rs := range resolved {
        skillsList = append(skillsList, SkillConfig{
            Name:        rs.Name,
            Description: rs.Description,
            Content:     rs.Content,
        })
    }
}
```

Import alias: `domainsrv "github.com/morphy76/tacito-square/internal/keeper/domain/service"`.

### 3. `SkillConfig` struct

`SkillConfig` (lines ~150-154) remains unchanged — it carries `Name`, `Description`, `Content`. The `ID` field is not needed in the propagated config.

### 4. Update `crd_coordinator_test.go`

Existing tests `TestResolveAndSynthesizeSystemPrompt_*` must be reviewed and extended:

- `TestResolveAndSynthesizeSystemPrompt_WithSkillCollection` — mock `ResolveCollectionSkills` for an agent that has a `SkillCollections` entry; assert skill content appears in the synthesized JSON.
- `TestResolveAndSynthesizeSystemPrompt_CollectionAndIndividual_Deduplicated` — agent has same skill ID in both collection and `Skills` slice; asserts it appears only once in `PropagatedAgentConfig.Skills`.
- `TestResolveAndSynthesizeSystemPrompt_SuspendedSkillOmitted` — mock returns `suspended` skill in collection; asserts it is absent from output.

All existing CRD coordinator tests must remain green.

## Acceptance Criteria

1. `ResolveAndSynthesizeSystemPrompt` calls `domainsrv.ResolveAgentSkills` (not a manual skill loop).
2. Agent with only `SkillCollections` (empty `Skills`) correctly populates `PropagatedAgentConfig.Skills`.
3. All 3 new CRD coordinator tests pass.
4. Existing `TestResolveAndSynthesizeSystemPrompt_*` tests remain green.
5. `crd_coordinator.go` does not import from `application/` layer — only from `domain/` and `application/ports/outbound/`.
