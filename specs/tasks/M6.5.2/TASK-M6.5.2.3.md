# TASK-M6.5.2.3: Database Repository — Soft Delete & List Filtering

| Field       | Value |
|-------------|-------|
| ID          | TASK-M6.5.2.3 |
| Status      | TODO |
| Spec        | SPEC-FR-M6.5.2 |
| Depends On  | TASK-M6.5.2.1 |

## Description

Modify the database repository `postgres.LLMBindingRepository` to:
1. Perform soft deletion (sets `status = inactive` and updates `updated_at`) instead of running a hard SQL `DELETE`.
2. Exclude `inactive` bindings from list results (`status <> 'inactive'`).

## Work Items

1. **RED Phase**:
   - Modify `internal/keeper/adapters/outbound/postgres/llm_binding_repository_test.go`:
     - Test that calling `Delete` sets the status to `inactive` instead of removing the row from the database.
     - Test that calling `List` does not return the soft-deleted binding in the results.
     - Test that we can create a new binding with the same name after the previous one was soft-deleted.

2. **GREEN Phase**:
   - Modify `internal/keeper/adapters/outbound/postgres/llm_binding_repository.go`:
     - Update the `Delete` method:
       ```sql
       UPDATE llm_bindings SET status = 'inactive', updated_at = $1 WHERE id = $2 AND tenant_id = $3
       ```
     - Update the `List` method to filter active bindings:
       ```sql
       SELECT ... FROM llm_bindings WHERE tenant_id = $1 AND status <> 'inactive' ORDER BY name ASC
       ```

3. **REFACTOR Phase**:
   - Ensure clean database context error handling and execution. Run tests using `go test ./internal/keeper/adapters/outbound/postgres/...`.

## Acceptance Criteria

1. Repository unit/integration tests compile and pass GREEN.
2. The `Delete` database call preserves the row but marks it `inactive`.
3. The `List` query successfully filters out `inactive` bindings.
