# BUG-M4.6: Echo Message Fails to Dispatch Due to NATS Subject Mismatch (Name vs UUID)

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M4.6                                                           |
| Status        | CLOSED                                                             |
| Severity      | HIGH                                                               |
| Milestone     | M4 — Operator Core                                                 |
| Affects       | internal/keeper/application/service/echo_service.go                 |
| Violates      | SPEC-FR-M4.8, SPEC-FR-M6.3                                         |
| Discovered    | Runtime testing of the community echo endpoint                     |

## Problem Statement

When the `/communities/:community_id/echo` endpoint is invoked, the request fails with a NATS timeout error: `"nats request to ts.community.sales-agents-net.agent.qa-agent: nats: no responders available for request"`. 

The root cause is a mismatch in how the NATS subject is constructed:
1. **Agent Pod Listener:** When the agent container is deployed, the Kubernetes Operator passes the community reference (`TS_AGENT_COMMUNITY_REF`) as the **community ID (UUID)** (e.g. `"0c8d02af-6235-4fae-825d-694c8547d09c"`). The agent pod's `EchoSubscriber` registers its subscription on the subject pattern `ts.community.{communityID}.agent.{agentName}` using this **community ID (UUID)**.
2. **Keeper Broadcaster:** Inside `EchoServiceImpl.EchoCommunity`, the Keeper constructs the broadcast subject by passing the community **Name** (`comm.Name`, e.g. `"sales-agents-net"`) instead of its **ID (UUID)** to `s.broadcaster.RequestEcho`. 

As a result, the Keeper broadcasts on the subject `ts.community.sales-agents-net.agent.qa-agent`, but the agent is listening on `ts.community.0c8d02af-6235-4fae-825d-694c8547d09c.agent.qa-agent`. Due to the subject mismatch, NATS reports that no responders are available.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| keeper / service | [echo_service.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/application/service/echo_service.go) | `EchoCommunity` passes `comm.Name` instead of `comm.ID.String()` to `s.broadcaster.RequestEcho`. |
| keeper / service / test | [echo_service_test.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/application/service/echo_service_test.go) | Mock expectations for `RequestEcho` incorrectly expect the community Name (`"test-comm"`) instead of the community ID (UUID). |

## Impact

1. **Echo Endpoint Failure:** The community echo capability is completely broken at runtime, rendering communication validation impossible.
2. **NATS Scale-from-Zero Failures:** Wildcard subscriptions meant to trigger scale-from-zero (SPEC-FR-M4.4) fail to catch messages because they are sent to the community Name subject instead of the community ID.

## Expected Behaviour

1. The Keeper's `EchoServiceImpl` MUST pass the **community ID (UUID)** as a string (`comm.ID.String()`) to the outbound `CommunityBroadcaster.RequestEcho` port.
2. The fanned-out NATS requests MUST be dispatched on the subject pattern `ts.community.{communityID_UUID}.agent.{agentName}`.

## Acceptance Criteria

1. Running a community echo request correctly targets the NATS subject built with the community ID (UUID), and receiving agents successfully handle and respond to the request.
2. Existing unit tests in `echo_service_test.go` are updated to expect the community ID (UUID) string as the second parameter of `RequestEcho` and pass cleanly.
