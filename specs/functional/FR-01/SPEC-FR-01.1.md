# SPEC-FR-01.1: Spawn Agent from Prompt + Skills

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-FR-01.1                       |
| Status        | VERIFIED                           |
| Milestone     | M1                                 |
| FR/NFR Ref    | FR-01.1                            |
| Component     | keeper                             |
| Depends On    | —                                  |

## Context

The Keeper must be able to create new agent instances from a prompt and skill set, deploying them as stateless containers in Kubernetes.

## Specification

1. `NewSpawnRequest(name, communityID, promptID, skillIDs, requestedBy)` MUST validate all required fields.
2. `NewAgentInstance(name, communityID, promptID, skillIDs)` MUST initialize the instance in `Pending` status.
3. `KeeperService.SpawnAgent(ctx, req)` MUST:
   a. Create the domain entity in `Pending` state
   b. Persist it via the `AgentStore` port
   c. Deploy via the `AgentSpawner` port
   d. Transition to `Running` on success
   e. Terminate with reason on spawn failure
4. Required fields: `name`, `communityID`, `promptID`. `skillIDs` MAY be empty.
5. `requestedBy` identifies the actor (user or agent) that triggered the spawn.

## Acceptance Criteria

1. `NewSpawnRequest` returns error if `name` is empty
2. `NewSpawnRequest` returns error if `communityID` is empty
3. `NewSpawnRequest` returns error if `promptID` is empty
4. `NewSpawnRequest` succeeds with valid inputs, generates UUID
5. `SpawnAgent` persists instance via store
6. `SpawnAgent` calls spawner
7. `SpawnAgent` transitions to Running on success
8. `SpawnAgent` terminates on spawn failure

## Files

- `internal/keeper/domain/spawn_request.go` ✅ IMPLEMENTED
- `internal/keeper/domain/agent_instance.go` ✅ IMPLEMENTED
- `internal/keeper/service/keeper_service.go` ✅ IMPLEMENTED
- `internal/keeper/domain/domain_test.go` ✅ Tests passing
- `internal/keeper/service/service_test.go` ✅ Tests passing
