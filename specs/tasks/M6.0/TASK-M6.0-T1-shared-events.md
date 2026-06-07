# TASK-M6.0-T1: Shared Event Package (`pkg/events`)

| Field       | Value                                 |
|-------------|---------------------------------------|
| Task ID     | TASK-M6.0-T1                          |
| Spec        | SPEC-FR-M6.0                          |
| Boundary    | Shared Package — `pkg/events`         |
| Status      | VERIFIED                              |
| Depends On  | —                                     |

## Objective

Create the shared package `pkg/events` containing the canonical `DomainEvent` envelope definition, the `NewDomainEvent` event builder constructor, and the conversational payload structs and constants.

## Files

| File | Action |
|------|--------|
| `pkg/events/event.go` | NEW |
| `pkg/events/conversational.go` | NEW |
| `pkg/events/event_test.go` | NEW |
| `pkg/events/conversational_test.go` | NEW |

## RED Phase

Create unit tests in `pkg/events/event_test.go` and `pkg/events/conversational_test.go`:

- `TestNewDomainEvent_Success`: Assert that constructor generates `EventID` (UUID v4), `OccurredAt` (valid RFC3339Nano UTC timestamp), correctly serializes payload, and returns `DomainEvent` by value.
- `TestNewDomainEvent_MissingFields`: Verify `NewDomainEvent` returns an error if `schemaRef`, `source`, or `tenantID` is empty.
- `TestConversationalPayloads_JSON`: Verify JSON marshalling/unmarshalling for `StartThreadPayload`, `AddUserMessagePayload`, `EndThreadPayload`, and `AgentResponsePayload` to confirm all fields match contracts (e.g. metadata is omitted if empty).

Run `make test` — must fail because package doesn't exist (RED).

## GREEN Phase

1. Create `pkg/events/event.go` implementing `DomainEvent` and `NewDomainEvent(...) (DomainEvent, error)` according to SPEC-FR-M6.0 Section 1.1 and 1.6.
2. Create `pkg/events/conversational.go` declaring URN constants and payload structs matching SPEC-FR-M6.0 Section 1.5, 2.1, 2.2, 2.3, and 5.6.

Run `make test` — tests must pass (GREEN).

## REFACTOR Phase

- Confirm `NewDomainEvent` returns value type to avoid unnecessary heap allocation.
- Verify `OccurredAt` timestamp is strictly UTC.
- Verify payload marshals properly and struct tagging is clean.
