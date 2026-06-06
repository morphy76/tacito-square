# TASK-M6.0-T3: Keeper Event Service Implementation

| Field       | Value                                                    |
|-------------|----------------------------------------------------------|
| Task ID     | TASK-M6.0-T3                                             |
| Spec        | SPEC-FR-M6.0                                             |
| Boundary    | Keeper Service — `internal/keeper/application/service`   |
| Status      | DRAFT                                                    |
| Depends On  | TASK-M6.0-T2                                             |

## Objective

Implement the `EventUseCase` and `EventStreamUseCase` service layer to handle business logic: validating/sanitizing events, mapping NATS subjects, and invoking outbound publishers/subscribers.

## Files

| File | Action |
|------|--------|
| `internal/keeper/application/service/event_service.go` | NEW |
| `internal/keeper/application/service/event_service_test.go` | NEW |
| `internal/keeper/application/service/echo_service.go` | DELETE |
| `internal/keeper/application/service/echo_service_test.go` | DELETE |

## RED Phase

Create unit tests in `internal/keeper/application/service/event_service_test.go` with mocked `EventPublisher` and `EventSubscriber`:

- `TestPublishEvent_Success`: Mock `EventPublisher.Publish`; publish a valid `start-thread` event; verify subject mapping, auto-population of `EventID`, `OccurredAt`, `TenantID`, and `Source` (`keeper/{instanceID}`).
- `TestPublishEvent_InvalidSchemaRef`: Assert validation failure if schemaRef doesn't match `urn:tacito:schema:*`.
- `TestPublishEvent_Sanitization`: Publish `add-user-message` containing control characters; assert they are stripped and the message is truncated to 4096.
- `TestPublishEvent_SanitizedEmpty`: Assert failure if message is empty after sanitization.
- `TestPublishEvent_SubjectRouting`: Assert that `urn:tacito:schema:conversational:*` routes to `ts.community.{communityID}.agent.{agentName}` while unknown schemas route to `ts.events.{tenantID}`.
- `TestSubscribeEvents`: Assert it calls `EventSubscriber.Subscribe` with the wildcard pattern `ts.community.>`.

Run `make test` — must fail (RED).

## GREEN Phase

1. Create `internal/keeper/application/service/event_service.go`:
   - Implement `EventServiceImpl` satisfying `inbound.EventUseCase` and `inbound.EventStreamUseCase`.
   - Implement input validation and sanitization using helper routines or standard unicode packages.
   - Inject outbound `EventPublisher` and `EventSubscriber` in constructor.
2. Delete `internal/keeper/application/service/echo_service.go` and its test.

Run `make test` — tests must pass (GREEN).

## REFACTOR Phase

- Ensure the service only depends on application port interfaces, not concrete NATS adapters.
- Confirm `keeper/{instanceID}` matches the hostname/machine name properly.
- Keep helper functions like sanitization clean and well-structured.
