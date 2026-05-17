# SPEC-FR-01.3: Config Snapshot at Spawn

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-FR-01.3                       |
| Status        | VERIFIED                           |
| Milestone     | M1                                 |
| FR/NFR Ref    | FR-01.3                            |
| Component     | keeper                             |
| Depends On    | SPEC-FR-01.1                       |

## Context

When an agent is spawned, its resolved configuration (prompt content, skill set, environment) must be captured as an immutable snapshot so that the agent's behavior is reproducible regardless of later prompt/skill changes.

## Specification

1. `AgentInstance.ConfigSnapshot` MUST be a `map[string]interface{}` populated at spawn time.
2. The snapshot MUST capture prompt ID, skill IDs, and any environment overrides.
3. Once set, the snapshot MUST NOT be modified during the agent's lifecycle.
4. The snapshot is initialized as an empty map in `NewAgentInstance`.

## Acceptance Criteria

1. New agent instance has an initialized (non-nil) ConfigSnapshot
2. Snapshot is persisted alongside the agent instance
3. Snapshot is immutable after creation (enforced by convention)

## Files

- `internal/keeper/domain/agent_instance.go` ✅ IMPLEMENTED (line 37, 62)
