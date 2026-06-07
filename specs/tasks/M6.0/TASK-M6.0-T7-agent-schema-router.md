# TASK-M6.0-T7: Agent Schema Router & Brain Handlers

| Field       | Value                                                   |
|-------------|---------------------------------------------------------|
| Task ID     | TASK-M6.0-T7                                            |
| Spec        | SPEC-FR-M6.0                                            |
| Boundary    | Agent Service — `internal/agent/application/service`    |
| Status      | IMPLEMENTED                                             |
| Depends On  | TASK-M6.0-T6                                            |

## Objective

Implement the `SchemaRouter` driving port to parse, validate, and dispatch incoming events based on their `schemaRef` to specific brain handlers (start-thread, add-user-message, and end-thread).

## Files

| File | Action |
|------|--------|
| `internal/agent/application/ports/inbound/schema_router.go` | NEW |
| `internal/agent/application/service/schema_router_impl.go` | NEW |
| `internal/agent/application/service/schema_router_impl_test.go` | NEW |

## RED Phase

Write unit tests in `internal/agent/application/service/schema_router_impl_test.go` with mocked `MessageProcessor`, `ShortTermMemory`, `LongTermMemory`, and NATS publisher:

- `TestSchemaRouter_StartThread`: Dispatch `start-thread`; verify it logs info and clears/initializes the Redis keys for `thread_id`.
- `TestSchemaRouter_AddUserMessage_Success`: Dispatch `add-user-message`; verify LLM processing completes successfully, assistant response is appended, and an `agent-response` event is published to `ts.community.{id}.agent.{name}.response` with headers.
- `TestSchemaRouter_AddUserMessage_LLMFailure_Rollback`: Mock `MessageProcessor` to fail; verify that `ShortTermMemory.RollbackLast` is called to revert the user turn, and a warning log is written. Assert that Redis delete failure doesn't cause a panic.
- `TestSchemaRouter_EndThread_LTM`: Dispatch `end-thread`; if LTM enabled, verify it retrieves thread history, compiles a summary, embeds it, persists it to Qdrant LTM, and clears STM.
- `TestSchemaRouter_UnknownSchemaRef`: Dispatch unregistered schemaRef URN; verify it logs `warn` and returns nil (silently dropped).

Run `make test` — must fail (RED).

## GREEN Phase

1. Create `internal/agent/application/ports/inbound/schema_router.go` defining the `SchemaRouter` interface.
2. Create `internal/agent/application/service/schema_router_impl.go`:
   - Implement `SchemaRouterImpl` satisfying `SchemaRouter`.
   - Setup handlers for `start-thread`, `add-user-message`, and `end-thread`.
   - Implement STM rollback logic on message processor failure.
   - Implement active `agent-response` event publication using `pkg/events.NewDomainEvent(...)`.
   - Implement summary consolidation to Qdrant LTM on `end-thread`.

Run `make test` — tests must pass (GREEN).

## REFACTOR Phase

- Ensure logging context includes `event_id`, `thread_id`, and `agent_name` for traceability.
- Double-check that `agent-response` uses the correct `agent/{agentID}` source identity.
- Verify Qdrant LTM integration handles errors gracefully.
