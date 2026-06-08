# BUG-M6.3: Inbound community events are routed to all spokes in hub-spoke topology instead of the Hub coordinator

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M6.3                                                           |
| Status        | OPEN                                                               |
| Severity      | HIGH                                                               |
| Milestone     | M6 — Communities & Messaging                                       |
| Affects       | `internal/keeper/application/service/event_service.go`, `internal/keeper/bootstrap.go` |
| Violates      | SPEC-FR-M6.1                                                       |
| Discovered    | During end-to-end community messaging verification where multiple spoke agents processed requests standalone and generated multiple answers/histories. |

## Problem Statement

In a `hub-spoke` community topology:
1. Inbound community events (such as conversational start-thread or add-user-message) are published by the BFF/Keeper via `EventServiceImpl.PublishEvent`.
2. When the event payload doesn't specify a target `agent_name` (standard for incoming user traffic), `EventServiceImpl.PublishEvent` defaults the NATS subject to `ts.community.{community_id}.agent.all`.
3. Since Spoke agents subscribe to both `ts.community.{community_id}.agent.{agent_name}` and `ts.community.{community_id}.agent.all`, all Spokes in the community receive the inbound event and process it standalone. This leads to the end user receiving multiple final responses and duplicate chat histories (e.g., three answers and three histories for a community with three spokes).
4. Meanwhile, the Hub agent (which only subscribes to `ts.community.{community_id}.agent.hub` to act as a centralized coordinator) never receives the event.

To comply with the SPEC-FR-M6.1 Hub-Spoke topology spec:
* "The BFF/Keeper publishes conversation events to the Hub's NATS subject: `ts.community.{community_id}.agent.hub`."
* `EventServiceImpl` must be aware of the community's topology. It must fetch the community from the database, check its topology, and route to `agent.hub` if the topology is `hub-spoke` and `agent_name` is not specified.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| EventServiceImpl | [event_service.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/application/service/event_service.go) | Lacks dependency on `CommunityRepository` and does not fetch community topology to determine the NATS routing subject (always routing to `.all` when `agent_name` is empty). |
| Keeper Bootstrap | [bootstrap.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/bootstrap.go) | Does not pass `CommunityRepository` into `NewEventService`. |
| Event Service Tests | [event_service_test.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/application/service/event_service_test.go) | Mock registry and test cases do not assert routing to the Hub subject for `hub-spoke` communities. |

## Impact

1. The Hub agent is bypassed entirely, breaking the hub-spoke orchestration state machine and coordinated network logic.
2. Multiple spoke agents execute redundant/parallel reasoning loops, leading to duplicate database records, multiple final answers sent to the client, and redundant/messy chat histories.
3. High resource usage on the LLM backend because all spoke agents execute concurrent reasoning loops for the same request.

## Expected Behaviour

1. The `EventServiceImpl` MUST depend on `outbound.CommunityRepository` (or check the community topology via database/cache).
2. When `PublishEvent` processes a conversational schema event (`urn:tacito:schema:conversational:*`) where `agent_name` is empty:
   * It MUST fetch the community definition using `community_id` from the repository.
   * If the community's topology is `hub-spoke`, the event MUST be published to NATS subject `ts.community.{community_id}.agent.hub`.
   * If the community's topology is `single-agent` (or any other topology), the event MUST be published to `ts.community.{community_id}.agent.all`.

## Acceptance Criteria

1. `CommunityRepository` is successfully added as a dependency to `EventServiceImpl` and wired up in Keeper's `bootstrap.go`.
2. A unit test in `event_service_test.go` is added to verify that publishing a conversational event without an `agent_name` for a `hub-spoke` community resolves the destination NATS subject to `ts.community.{community_id}.agent.hub`.
3. A unit test in `event_service_test.go` verifies that for a `single-agent` community, the destination NATS subject resolves to `ts.community.{community_id}.agent.all`.
4. All unit/integration tests run and pass.
