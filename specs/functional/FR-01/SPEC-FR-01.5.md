# SPEC-FR-01.5: Terminate Agents

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-FR-01.5                       |
| Status        | VERIFIED                           |
| Milestone     | M1                                 |
| FR/NFR Ref    | FR-01.5                            |
| Component     | keeper                             |
| Depends On    | SPEC-FR-01.1, SPEC-FR-01.2        |

## Context

Agents must be terminable by API consumers or internal rules. Termination transitions the agent to a terminal state and destroys the K8s deployment.

## Specification

1. `AgentInstance.Terminate(reason)` MUST:
   a. Call `TransitionTo(Terminated)` (valid from `Running` or `Degraded`)
   b. Set `TerminationReason` on success
   c. Return error if already terminated
2. `KeeperService.TerminateAgent(ctx, id, reason)` MUST:
   a. Find the agent by ID
   b. Call `instance.Terminate(reason)`
   c. Persist the updated state
   d. Call `spawner.Destroy(ctx, id)` to remove the K8s deployment
3. `DELETE /api/v1/agents/:id` MUST accept optional `{"reason": "..."}` body.
4. Default reason when body is empty: `"terminated via API"`.
5. Returns 204 No Content on success.

## Acceptance Criteria

1. Terminate from Running succeeds with reason recorded
2. Terminate from Degraded succeeds
3. Terminate from Terminated returns error
4. Terminate from Pending returns error (Pending → Terminated is valid in state machine)
5. K8s deployment is destroyed on termination
6. HTTP 204 returned on success

## Files

- `internal/keeper/domain/agent_instance.go` ✅ IMPLEMENTED (Terminate method)
- `internal/keeper/service/keeper_service.go` ✅ IMPLEMENTED (TerminateAgent)
- `internal/keeper/adapters/inbound/httphandler/handler.go` ✅ IMPLEMENTED (DELETE route)
