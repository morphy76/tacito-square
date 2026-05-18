# SPEC-FR-03.1: PostgreSQL AgentStore and Migrations

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-FR-03.1                      |
| Status        | DRAFT                              |
| Milestone     | M3                                 |
| FR/NFR Ref    | FR-01.1                            |
| Component     | keeper                             |
| Depends On    | SPEC-FR-01.1                       |

## Context

For the M3 deployable core, the Keeper must persist `agent_instance` state in PostgreSQL to survive restarts. This requires establishing the base PostgreSQL connection strategy (`pgxpool`), the schema for `agent_instance`, and a robust migration pipeline using `goose` deployed via a Kubernetes Job.

## Specification

### 1. Data Model & Schema

Based on `SPEC-ARCH-002`, the schema must track agent instances and their associated skills using a strict relational model for data integrity.

```sql
-- db/migrations/00001_init.sql
-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS agent_instance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    community_id UUID NOT NULL,
    prompt_id UUID NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'Pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS agent_instance_skill (
    agent_instance_id UUID NOT NULL REFERENCES agent_instance(id) ON DELETE CASCADE,
    skill_id UUID NOT NULL,
    PRIMARY KEY (agent_instance_id, skill_id)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS agent_instance_skill;
DROP TABLE IF EXISTS agent_instance;
-- +goose StatementEnd
```

### 2. PostgreSQL Client Configuration

The Keeper application MUST use `pgxpool` (`github.com/jackc/pgx/v5/pgxpool`) with the following behavior:
- **Connection Configuration:** Configurable via environment variables (e.g., `DB_MAX_CONNS`, `DB_MIN_CONNS`, `DB_MAX_CONN_LIFETIME`).
- **Resiliency:** The database connection initialization at application startup MUST implement exponential backoff retries. The database pod might become ready moments after the Keeper pod; Keeper MUST not crash loop immediately.
- **Context Usage:** All `AgentStore` repository methods MUST accept and propagate `context.Context` to database queries to honor HTTP deadlines and cancellations.

### 3. Migration Strategy (`goose`)

Migrations MUST be handled separately from the application lifecycle to avoid race conditions and restrict database privileges.

- **Tooling:** Use `goose` (`github.com/pressly/goose/v3`).
- **Storage:** Store `.sql` files in the `db/migrations/` directory.
- **Production Execution:** Helm charts MUST include a K8s `Job` annotated with `helm.sh/hook: pre-install,pre-upgrade`. This job will execute the `goose up` command against the target database before the Keeper Deployment is rolled out.
- **Local Execution:** A Makefile target `make db-migrate` MUST be provided for developer convenience using a local `goose` binary or a Docker container.

## Acceptance Criteria

1. Schema `00001_init.sql` is successfully applied to a clean PostgreSQL instance.
2. A Keeper `agent_instance` domain object can be saved and retrieved accurately, including its mapped skills in `agent_instance_skill`.
3. Keeper startup retries connection correctly when the database is initially unreachable.
4. Deployment via Helm successfully runs the `pre-install` Job, applying the migration, before the Keeper pod starts.
5. `goose down` correctly tears down the schema without orphaned objects.
