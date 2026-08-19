# BUG-M6.5: Hub Agent Deployed with Role 'spoke' in Hub-Spoke Community

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M6.5                                                           |
| Status        | CLOSED                                                             |
| Severity      | HIGH                                                               |
| Milestone     | M6 — Communities & Messaging                                       |
| Affects       | `internal/keeper/adapters/outbound/crd/crd_coordinator.go`          |
| Violates      | SPEC-FR-M6.1                                                       |
| Discovered    | Reported when assigning a hub agent to a hub-spoke community.       |

## Problem Statement

When assigning a Hub agent to a Hub-Spoke community, the agent is deployed with its environment role set to `"spoke"`. 

The root cause is in `internal/keeper/adapters/outbound/crd/crd_coordinator.go` inside the `SubmitAgentCRD` method. When constructing a new `v1alpha1.TacitoAgent` Custom Resource or updating an existing one, the `Role` field of `v1alpha1.TacitoAgentSpec` is never populated from `agent.Role`. Since the field is empty, the Operator defaults it to `"spoke"`.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| CRD Coordinator | [crd_coordinator.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/outbound/crd/crd_coordinator.go) | Missing mapping for `Spec.Role` on both the Create and Update paths in `SubmitAgentCRD`. |
| CRD Coordinator Tests | [crd_coordinator_test.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/outbound/crd/crd_coordinator_test.go) | Missing test cases verifying that `agent.Role` (both "hub" and "spoke") is correctly propagated to the CRD. |

## Impact

1. A Hub agent behaves as a Spoke agent, rendering the Hub-Spoke orchestration state machine and NATS routing logic non-functional because the container runs with `TS_AGENT_ROLE=spoke`.
2. Asynchronous orchestration flow fails, since no agent in the community listens on the `agent.hub` subject or manages the state.

## Expected Behaviour

1. When calling `SubmitAgentCRD` for a new agent, the generated CRD's `Spec.Role` MUST match the agent's `Role`.
2. When updating an existing agent CRD, the `Spec.Role` MUST be updated to match the agent's `Role`.

## Acceptance Criteria

1. In `internal/keeper/adapters/outbound/crd/crd_coordinator.go`, `SubmitAgentCRD` successfully propagates the `agent.Role` field to `Spec.Role` on both the create and update paths.
2. Unit tests in `internal/keeper/adapters/outbound/crd/crd_coordinator_test.go` assert that `agent.Role` is correctly propagated to the CRD spec for both `"hub"` and `"spoke"`.
