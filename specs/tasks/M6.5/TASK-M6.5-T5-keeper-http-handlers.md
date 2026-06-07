# TASK-M6.5-T5: Keeper HTTP Handlers & Public Caching

| Field       | Value                                 |
|-------------|---------------------------------------|
| Task ID     | TASK-M6.5-T5                          |
| Spec        | SPEC-FR-M6.5                          |
| Boundary    | Keeper HTTP Handlers                  |
| Status      | DRAFT                                 |
| Depends On  | TASK-M6.5-T4                          |

## Objective

Build the community-scoped REST endpoints in the Keeper HTTP server to serve individual Agent Cards, collective Community Cards, and indexes, appending public Cache-Control headers to support proxy caching.

## Files

| File | Action |
|------|--------|
| `internal/keeper/adapters/inbound/http/card_handlers.go` | NEW |
| `internal/keeper/adapters/inbound/http/card_handlers_test.go` | NEW |
| `internal/keeper/bootstrap.go` | MODIFY |

## RED Phase

1. Create handler tests in `internal/keeper/adapters/inbound/http/card_handlers_test.go` using `gin.SetMode(gin.TestMode)`.
2. Issue dynamic HTTP requests to:
   - `GET /api/v1/communities/:community_id/agents/:agent_id/.well-known/agent-card.json`
   - `GET /api/v1/communities/:community_id/.well-known/community-card.json`
   - `GET /api/v1/communities/:community_id/.well-known/agent-cards.json`
3. Assert that the responses return HTTP 200, valid JSON schemas matching specifications, and the header `Cache-Control` equal to `public, max-age=30`.
4. Verify tests fail to compile or route (RED).

## GREEN Phase

1. Create `internal/keeper/adapters/inbound/http/card_handlers.go` implementing the request handling logic.
2. In the handlers:
   - Fetch target entities (agent, community) from repositories/DB.
   - Enforce tenant isolation utilizing headers/context.
   - Return appropriate HTTP 404/500 errors formatted as `{ "error": "Clear error msg" }` on validation or database issues.
   - Write headers: `c.Header("Cache-Control", "public, max-age=30")`.
3. Register the new HTTP routes in `internal/keeper/bootstrap.go` inside the `v1` path group.
4. Verify that all tests pass (GREEN).

## REFACTOR Phase

- Ensure that middlewares (TenantResolutionMiddleware, DatabaseAvailabilityMiddleware) intercept the routes correctly.
- Review Gin route definitions to prevent overlapping or duplicate routing patterns.
