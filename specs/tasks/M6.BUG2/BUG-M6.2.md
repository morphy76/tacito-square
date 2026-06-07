# BUG-M6.2: Unassigning agent from community does not evict registration or cards cache

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M6.2                                                           |
| Status        | CLOSED                                                             |
| Severity      | HIGH                                                               |
| Milestone     | M6 — Communities & Messaging                                       |
| Affects       | `internal/keeper/application/service/agent_service.go`, `internal/keeper/adapters/outbound/postgres/agent_repository.go` |
| Violates      | SPEC-FR-M6.5                                                       |
| Discovered    | Manual verification that unassigned agents are still present in community-card.json API |

## Problem Statement

When an Agent is unassigned from a Community (via `DELETE /api/v1/communities/:community_id/agents/:agent_id` in Keeper):
1. The agent's community reference is updated in the `agents` table, but its active registration row is left in the `agent_registrations` table.
2. The agent's card is not invalidated/removed from the Redis active cache.
3. No offline status event is broadcast to NATS for the unassigned agent.
4. As a result, endpoints like `/api/v1/communities/:community_id/.well-known/community-card.json` still serve the unassigned agent card, violating the expectation that it should not be present after unassignment.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| AgentService | `internal/keeper/application/service/agent_service.go` | `Unassign` does not delete the DB registration, evict cache keys, or broadcast status updates. |
| AgentRepository Port | `internal/keeper/application/ports/outbound/repositories.go` | Missing `DeleteRegistration` driving interface. |
| Postgres Agent Repository | `internal/keeper/adapters/outbound/postgres/agent_repository.go` | Missing implementation of `DeleteRegistration`. |

## Impact

1. Outdated and unassigned agent cards persist indefinitely in the community registry until a manual DB delete or prune loop occurs.
2. Discovery lists still show agents that are no longer assigned to communities, causing capability routing failures when messages are sent to unassigned/offline agents.

## Expected Behaviour

1. When an agent is unassigned from a community, the associated registration row MUST be deleted from the `agent_registrations` table.
2. The cache keys `communities:<community_id>:agents:<agent_id>` and `communities:<community_id>:registry` MUST be invalidated/removed from Redis.
3. A NATS status update event with payload `{"status":"offline"}` MUST be published to `ts.community.<community_id>.agent.<agent_id>.status`.

## Acceptance Criteria

1. A unit test in `agent_service_test.go` asserts that unassignment invokes `DeleteRegistration` on the repository, invalidates the registry/agent cache keys, and publishes the offline status event (RED Phase).
2. The repository, port interface, service, and bootstrap are updated to perform registration deletion, cache eviction, and NATS event publication (GREEN Phase).
3. The tests pass successfully.
