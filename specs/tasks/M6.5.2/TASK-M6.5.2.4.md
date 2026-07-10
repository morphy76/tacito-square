# TASK-M6.5.2.4: Application Service Layer — Redis Caching Integration

| Field       | Value |
|-------------|-------|
| ID          | TASK-M6.5.2.4 |
| Status      | TODO |
| Spec        | SPEC-FR-M6.5.2 |
| Depends On  | TASK-M6.5.2.2, TASK-M6.5.2.3 |

## Description

Integrate the Redis cache adapter (`sharedports.Cache`) into the `LLMBindingService` application layer to enable read-through caching on get operations and eviction logic on updates and deletions.

## Work Items

1. **RED Phase**:
   - Create the unit test file `internal/keeper/application/service/llm_binding_service_test.go`:
     - Test that calling `GetByID` successfully retrieves the item from the cache if it exists (cache hit).
     - Test that calling `GetByID` queries PostgreSQL on cache miss, caches the item, and returns it.
     - Test that calling `Update` or `Delete` invalidates the cached binding under the key `{tenant_id}:llm-bindings:{id}`.

2. **GREEN Phase**:
   - Modify `internal/keeper/application/service/llm_binding_service.go`:
     - Update the struct `LLMBindingService` to include `cache sharedports.Cache`.
     - Update the constructor `NewLLMBindingService` to accept `cache sharedports.Cache`.
     - Implement key naming logic: `{tenant_id}:llm-bindings:{id}`.
     - Implement caching in `GetByID`: check cache first, then database, then populate cache with TTL 300 seconds.
     - Implement cache eviction in `Update` and `Delete` by calling `cache.Invalidate(ctx, key)`.
   - Modify `internal/keeper/bootstrap.go`:
     - Pass the `cacheClient` dependency when instantiating `llmService`.

3. **REFACTOR Phase**:
   - Verify proper error handling: cache errors must be logged as warnings but must not cause the business operation to fail (graceful degradation).

## Acceptance Criteria

1. Service tests in `llm_binding_service_test.go` pass successfully.
2. Caching is fully integrated for `GetByID`, `Update`, and `Delete` operations.
3. Keeper bootstrap compiles and wires dependencies cleanly.
