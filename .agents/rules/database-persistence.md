---
trigger: glob
globs: ["**/*.go", "**/*.sql"]
description: Relational database repository standards (pgx/v5) and SQL migration conventions (goose/v3).
---

# Database & Persistence Guidelines (PostgreSQL & Goose)

This rule establishes strict standards for database access, transaction management, query design, and SQL schema migrations using **PostgreSQL (`github.com/jackc/pgx/v5`)** and **Goose (`github.com/pressly/goose/v3`)**.

## 1. Locked Stack & Prohibition of ORMs

- **Approved Driver**: `github.com/jackc/pgx/v5` and `pgxpool`.
- **Approved Migration Tool**: `github.com/pressly/goose/v3`.
- **STRICT PROHIBITION**: Never use GORM, Ent, or other dynamic ORM libraries. All database queries must use explicit, parameterized SQL statements.

## 2. Repository Pattern & Port Implementations

All database persistence must live inside outbound adapters (`internal/<component>/adapters/outbound/postgres/`):

- **Interface Implementation**: Repositories must implement outbound port interfaces defined in `application/ports/outbound/`.
- **Domain Mapping**: Map database rows into pure domain model structs upon retrieval. Never return database-specific row structs or SQL driver types to application services.
- **Parametrized Queries**: Always use positional parameter placeholders (`$1, $2, $3`). Never concatenate raw strings or format unescaped values into SQL queries.
- **Domain Error Translation**: Map database driver errors to domain sentinel errors:
  ```go
  if errors.Is(err, pgx.ErrNoRows) {
      return nil, errors.NewNotFound("agent not found", id.String())
  }
  ```

## 3. Transaction Management (`pgx.Tx`)

- **Context Propagation**: Always pass the incoming `ctx context.Context` to every query, exec, or transaction operation.
- **Explicit Rollbacks**: Always defer a rollback immediately upon beginning a transaction; `tx.Commit(ctx)` will supersede the rollback on success:
  ```go
  tx, err := pool.Begin(ctx)
  if err != nil {
      return fmt.Errorf("begin tx: %w", err)
  }
  defer func() { _ = tx.Rollback(ctx) }()

  // Execute queries with tx...

  if err := tx.Commit(ctx); err != nil {
      return fmt.Errorf("commit tx: %w", err)
  }
  return nil
  ```

## 4. Goose SQL Schema Migrations

All database schema modifications must be recorded as Goose SQL migration files:

- **Location**: Store migrations in `internal/keeper/adapters/outbound/postgres/migrations/` (or component migration directory).
- **Naming Convention**: Use sequential timestamped filenames: `<YYYYMMDDHHMMSS>_<short_description>.sql` (e.g. `20260601120000_create_agents_table.sql`).
- **Idempotent Up and Down Blocks**: Every migration MUST include both `-- +goose Up` and `-- +goose Down` sections:
  ```sql
  -- +goose Up
  CREATE TABLE IF NOT EXISTS agents (
      id UUID PRIMARY KEY,
      tenant_id VARCHAR(64) NOT NULL,
      name VARCHAR(128) NOT NULL,
      status VARCHAR(32) NOT NULL,
      created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
      updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
  );
  CREATE INDEX IF NOT EXISTS idx_agents_tenant ON agents(tenant_id);

  -- +goose Down
  DROP TABLE IF EXISTS agents;
  ```
- **Tenant Scoping & Indexes**: Always include composite indexes on `tenant_id` for multi-tenant query efficiency.

---

## Developer Checklists & Verifications

- [ ] Are all queries parameterized with `$1, $2, ...`?
- [ ] Is `pgx.ErrNoRows` mapped to a domain error instead of returned raw?
- [ ] Are transactions wrapped in deferred rollbacks?
- [ ] Does every new migration file have both `-- +goose Up` and `-- +goose Down` blocks?
