# TASK-M6.0-T9: Bootstrap Wiring, Echo Cleanup, and Verification

| Field       | Value                                                   |
|-------------|---------------------------------------------------------|
| Task ID     | TASK-M6.0-T9                                            |
| Spec        | SPEC-FR-M6.0                                            |
| Boundary    | Bootstrap & Wiring — `cmd/`, `internal/keeper`, `internal/agent` |
| Status      | VERIFIED                                                |
| Depends On  | TASK-M6.0-T5, TASK-M6.0-T8                              |

## Objective

Clean up the deprecated echo feature code, rewire keeper and agent bootstrap entrypoints to use the new event-driven pipeline, and verify that the monorepo builds and passes all tests successfully.

## Files

| File | Action |
|------|--------|
| `internal/keeper/bootstrap.go` | MODIFY |
| `internal/agent/bootstrap.go` | MODIFY |
| `cmd/agent/main.go` | MODIFY |
| `cmd/keeper/main.go` | MODIFY |
| Delete remaining echo files (see below) | DELETE |

## RED Phase

Verify that all echo infrastructure files listed under SPEC-FR-M6.0 Section 6 are removed or can no longer be compiled:

- Confirm delete of:
  - `internal/keeper/domain/model/echo.go`
  - `internal/keeper/domain/model/echo_test.go`
  - `internal/keeper/application/service/echo_service.go`
  - `internal/keeper/application/service/echo_service_test.go`
  - `internal/keeper/adapters/outbound/nats/community_broadcaster.go`
  - `internal/keeper/adapters/outbound/nats/community_broadcaster_test.go`
  - `internal/keeper/adapters/inbound/http/echo_handlers.go`
  - `internal/keeper/adapters/inbound/http/echo_handlers_test.go`
  - `internal/agent/adapters/inbound/nats/echo_subscriber.go`
  - `internal/agent/adapters/inbound/nats/echo_subscriber_test.go`
- Try compiling the repository with `make build` — should fail if any imports remain (RED).

## GREEN Phase

1. Modify `internal/keeper/bootstrap.go` to remove echo handler registration and wire up `NATSEventPublisher`, `NATSEventSubscriber`, `EventServiceImpl`, and HTTP handlers.
2. Modify `internal/agent/bootstrap.go` to remove `EchoSubscriber` exports and add new generic subscriber setup functions.
3. Modify `cmd/agent/main.go` to inject `SchemaRouterImpl` handlers (conversational start/add-message/end) and wire NATS subscription.
4. Modify `cmd/keeper/main.go` if needed to inject required configurations (e.g. `keeper.sse.heartbeat_seconds` default).
5. Delete the deprecated echo source files.

Run `make build` and `make test` — all builds and tests must pass successfully (GREEN).

## REFACTOR Phase

- Ensure there are no leftover imports, comments, or stubs of the echo system.
- Confirm metrics (`keeper_sse_active_connections`, `keeper_events_published_total`, etc.) are registered correctly in bootstrap.
