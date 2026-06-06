# TASK-M6.0-T5: Keeper HTTP Handlers & OpenAPI Spec

| Field       | Value                                                   |
|-------------|---------------------------------------------------------|
| Task ID     | TASK-M6.0-T5                                            |
| Spec        | SPEC-FR-M6.0                                            |
| Boundary    | Keeper HTTP Layer — `internal/keeper/adapters/inbound/http` |
| Status      | DRAFT                                                   |
| Depends On  | TASK-M6.0-T4                                            |

## Objective

Implement HTTP route handlers for event publication and real-time streaming, register them under the `v1` API group, remove echo endpoints, and update the OpenAPI contract.

## Files

| File | Action |
|------|--------|
| `internal/keeper/adapters/inbound/http/event_handlers.go` | NEW |
| `internal/keeper/adapters/inbound/http/event_handlers_test.go` | NEW |
| `internal/keeper/adapters/inbound/http/echo_handlers.go` | DELETE |
| `internal/keeper/adapters/inbound/http/echo_handlers_test.go` | DELETE |
| `internal/keeper/openapi.json` | MODIFY |

## RED Phase

Write Gin-based HTTP handler tests in `internal/keeper/adapters/inbound/http/event_handlers_test.go`:

- `TestPublishEventHandler_Success`: Mock `EventUseCase`; verify `POST /api/v1/events` receives partial JSON, invokes usecase, and returns `202 Accepted` with full event payload.
- `TestPublishEventHandler_InputOmission`: Verify that input fields like `event_id` or `tenant_id` sent by client in request body are ignored (not rejected) and overwritten by the system.
- `TestPublishEventHandler_ValidationErrors`: Verify that invalid JSON payload returns `400 Bad Request`, and invalid schemaRef format (not starting with `urn:tacito:schema:*`) returns `422 Unprocessable Entity`.
- `TestStreamEventsHandler_SSE`: Mock `EventStreamUseCase`; call `GET /api/v1/events/stream`; assert headers are set (`text/event-stream`), heartbeat comments are sent, and events are formatted as standard SSE frames.
- `TestStreamEventsHandler_Disconnect`: Verify that client disconnection context cancellation causes NATS subscription to call `Stop()` and terminate the handler goroutine.

Run `make test` — must fail (RED).

## GREEN Phase

1. Create `internal/keeper/adapters/inbound/http/event_handlers.go`:
   - Implement `EventHandler` with `PublishEvent(c *gin.Context)` and `StreamEvents(c *gin.Context)`.
   - Setup Gin routing block and content-type parsing.
   - Implement SSE writer stream logic with keep-alive heartbeats using configurable Viper value `keeper.sse.heartbeat_seconds` (default: 15).
2. Modify `internal/keeper/openapi.json`:
   - Delete echo pathways.
   - Declare tags `events/publish` and `events/stream`.
   - Add specifications for `POST /api/v1/events` and `GET /api/v1/events/stream`.
3. Delete `internal/keeper/adapters/inbound/http/echo_handlers.go` and its test.

Run `make test` — tests must pass (GREEN).

## REFACTOR Phase

- Confirm unified error JSON formatting is used: `{"error": "message"}` on any failure.
- Ensure Gin context is correctly managed, using non-blocking writes and graceful shutdown loops.
- Verify `c.Writer.Flush()` is called after each event block to guarantee immediate delivery.
