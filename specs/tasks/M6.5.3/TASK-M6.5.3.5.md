# TASK-M6.5.3.5: Caching & Inbound HTTP Handlers — CRUD, Association, and Resolution APIs

| Field       | Value |
|-------------|-------|
| ID          | TASK-M6.5.3.5 |
| Status      | IMPLEMENTED |
| Spec        | SPEC-FR-M6.5.3 |
| Depends On  | TASK-M6.5.3.4 |

## Description

Implement redis caching for the resolved prompts per agent, and create/update Gin HTTP controllers for prompt versioning, collection membership, and agent association.

## Work Items

1. **RED Phase**:
   - In `internal/keeper/adapters/inbound/http/prompt_handlers_test.go`, write unit tests:
     - Verify status=draft on create, transitioning to active/archived on PUT.
     - Verify conflict error 409 is returned when adding a duplicate member to a collection.
     - Verify HTTP endpoints for attaching/detaching prompts and collections from agents.
     - Verify `GET /api/v1/agents/{id}/prompts` returns the resolved prompts list, and that caching/invalidation works (mock Redis cache calls).

2. **GREEN Phase**:
   - **Caching Implementation**:
     - Create `internal/keeper/adapters/outbound/redis/prompt_cache.go` (or update existing redis cache logic).
     - Store the resolved prompt list under the key format: `agent-prompts:{tenantID}:{agentID}` with the configured TTL (Viper key `cache.agent_prompts_ttl`).
     - hook cache invalidation on any agent-prompt / collection attachment changes, status changes, or collection membership updates.
   - **Gin Handlers (HTTP Controllers)**:
     - Modify `internal/keeper/adapters/inbound/http/prompt_handlers.go`:
       - Handle status defaults on create.
       - Implement version updates on PUT when content changes.
     - Create/Update prompt collection membership and agent attachment endpoints:
       - `POST /api/v1/prompt-collections/{id}/prompts/{prompt_id}`
       - `DELETE /api/v1/prompt-collections/{id}/prompts/{prompt_id}`
       - `POST /api/v1/agents/{id}/prompts/{prompt_id}`
       - `DELETE /api/v1/agents/{id}/prompts/{prompt_id}`
       - `POST /api/v1/agents/{id}/prompt-collections/{collection_id}`
       - `DELETE /api/v1/agents/{id}/prompt-collections/{collection_id}`
       - `GET /api/v1/agents/{id}/prompts`
     - Register the routes in `internal/keeper/bootstrap.go`.

3. **REFACTOR Phase**:
   - Run handlers integration tests: `go test ./internal/keeper/adapters/inbound/http/...` to ensure all tests pass GREEN.

## Acceptance Criteria

1. Endpoints match the API contract routes and status code rules.
2. Draft prompt creation defaults correctly.
3. Content changes result in version record creation.
4. Cache handles resolved lists, with invalidation triggers running on modifications.
