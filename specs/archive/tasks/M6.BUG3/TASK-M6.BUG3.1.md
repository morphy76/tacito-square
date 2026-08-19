# TASK-M6.BUG3.1: Topology-Aware Event Routing in EventServiceImpl

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M6.BUG3.1                              |
| Status        | VERIFIED                                    |
| Spec          | BUG-M6.3                                    |
| Depends On    | none                                        |

## Description

The `EventServiceImpl.PublishEvent` in the Keeper application service currently routes all inbound conversational events with an empty `agent_name` to `ts.community.{community_id}.agent.all`. This is correct for `single-agent` communities (where the sole spoke agent subscribes to `agent.all`), but violates the Hub-Spoke topology contract defined in SPEC-FR-M6.1.

In a `hub-spoke` community, the Hub agent is the sole entry point and only subscribes to `ts.community.{community_id}.agent.hub`. Broadcasting to `agent.all` causes all spoke agents to independently process the message, producing duplicate responses and histories.

**Fix**: Add `CommunityRepository` as a dependency to `EventServiceImpl`. When a conversational event has an empty `agent_name`, look up the community topology. If `hub-spoke`, route to `agent.hub`. If `single-agent`, route to `agent.all`. Additionally, the existing test `TestPublishEvent_Success_Conversational_OptionalAgentName` must be updated to reflect the new dependency signature and topology-based routing.

### Scope

This task covers a single logical boundary: **Keeper Application Service** (`event_service.go`) and its bootstrap wiring (`bootstrap.go`).

## Work Items

1. **RED Phase**:
   - Add a `mockCommunityRepository` to `event_service_test.go` implementing `outbound.CommunityRepository`.
   - Add test `TestPublishEvent_HubSpoke_RoutesToHub`: publishes a conversational event with empty `agent_name` for a `hub-spoke` community; asserts the NATS subject is `ts.community.{community_id}.agent.hub`.
   - Add test `TestPublishEvent_SingleAgent_RoutesToAll`: publishes a conversational event with empty `agent_name` for a `single-agent` community; asserts the NATS subject is `ts.community.{community_id}.agent.all`.
   - Add test `TestPublishEvent_UnknownCommunity_RoutesToAll`: publishes a conversational event with an unknown/invalid `community_id`; asserts graceful degradation by routing to `agent.all`.
   - Update existing test `TestPublishEvent_Success_Conversational_OptionalAgentName` to supply a mock community repository (single-agent topology).
   - Update all other existing `NewEventService(pub, sub)` call sites in the test file to include the new `commRepo` parameter.
   - All new tests must fail (RED) because `NewEventService` doesn't accept a community repository yet.

2. **GREEN Phase**:
   - Modify `EventServiceImpl` struct to hold a `commRepo outbound.CommunityRepository` field.
   - Modify `NewEventService` signature to accept `commRepo outbound.CommunityRepository`.
   - In `PublishEvent`, when `routeInfo.AgentName == ""`:
     - Parse `routeInfo.CommunityID` as `uuid.UUID`.
     - Call `commRepo.GetByID(ctx, communityUUID)` to fetch the community.
     - If the community's `Topology == model.CommunityTopologyHubSpoke`, set subject to `ts.community.{community_id}.agent.hub`.
     - If the community's topology is `single-agent`, or if the lookup fails (graceful degradation), set subject to `ts.community.{community_id}.agent.all`.
   - Update `bootstrap.go` to pass `communityRepo` into `NewEventService`.
   - All tests must pass (GREEN).

3. **REFACTOR Phase**:
   - Ensure debug/info logging of the topology-based routing decision.
   - Verify no import violations (domain/application boundary integrity).

## Acceptance Criteria

1. All new and updated unit tests in `event_service_test.go` pass.
2. All existing unit tests in the `service_test` package continue to pass with the updated `NewEventService` signature.
3. The `EventServiceImpl` routes hub-spoke community events without an `agent_name` to `ts.community.{community_id}.agent.hub`.
4. The `EventServiceImpl` routes single-agent community events without an `agent_name` to `ts.community.{community_id}.agent.all`.
5. Graceful degradation: if the community lookup fails (unknown ID, parse error), events fall back to `ts.community.{community_id}.agent.all`.
6. `bootstrap.go` correctly passes `communityRepo` into `NewEventService`.
