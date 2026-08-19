# TASK-M6.5.2.1: DB Migration — Soft Delete Unique Index

| Field       | Value |
|-------------|-------|
| ID          | TASK-M6.5.2.1 |
| Status      | IMPLEMENTED |
| Spec        | SPEC-FR-M6.5.2 |
| Depends On  | none |

## Description

Create a new goose database migration to replace the physical unique constraint `unique_llm_bindings_tenant_name` with a partial unique index filtering out inactive status records. This allows tenants to reuse the name of a soft-deleted (inactive) binding when creating a new active one.

## Work Items

1. **Migration Creation**:
   - Create the migration file under `deploy/postgres/migrations/00003_llm_bindings_soft_delete_index.sql`.

2. **Migration Statements (`Up`)**:
   - Write SQL to drop the existing unique constraint:
     ```sql
     ALTER TABLE llm_bindings DROP CONSTRAINT IF EXISTS unique_llm_bindings_tenant_name;
     ```
   - Write SQL to create a partial unique index on active bindings:
     ```sql
     CREATE UNIQUE INDEX unique_llm_bindings_tenant_name_active 
     ON llm_bindings (tenant_id, name) 
     WHERE status <> 'inactive';
     ```

3. **Migration Statements (`Down`)**:
   - Write rollback SQL to drop the partial index:
     ```sql
     DROP INDEX IF EXISTS unique_llm_bindings_tenant_name_active;
     ```
   - Re-add the physical unique constraint:
     ```sql
     ALTER TABLE llm_bindings ADD CONSTRAINT unique_llm_bindings_tenant_name UNIQUE (tenant_id, name);
     ```

4. **Verification**:
   - Apply the migration and rollback tests against the local database schema to verify they run cleanly.

## Acceptance Criteria

1. Migration `00003_llm_bindings_soft_delete_index.sql` compiles and runs successfully using the goose CLI or make commands.
2. Rolling back the migration (Down) restores the original UNIQUE constraint correctly.
3. The database constraint behaves as expected: duplicate names are forbidden for active records, but allowed if the older record is inactive.
