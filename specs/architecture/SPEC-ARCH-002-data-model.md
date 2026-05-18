# SPEC-ARCH-002: Keeper Data Model

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-ARCH-002                      |
| Status        | ACCEPTED                           |
| Component     | keeper                             |

## Specification

The Keeper PostgreSQL data model MUST implement the following entities:

### Entities

| Entity | PK | Relationships |
|--------|-----|---------------|
| `agent_instance` | uuid | belongs_to community, uses prompt, has_many skills |
| `prompt` | uuid | has_many prompt_versions |
| `prompt_version` | uuid | belongs_to prompt |
| `skill` | uuid | has_many skill_tools |
| `skill_tool` | uuid | belongs_to skill |
| `community` | uuid | has_many threads, has_many agent_instances |
| `thread` | uuid | belongs_to community, has_many hitl_callbacks |
| `hitl_callback` | uuid | belongs_to thread, belongs_to agent_instance |
| `external_source` | uuid | — |
| `audit_event` | uuid | references agent_instance |

### State Machines

- `agent_instance.status`: `Pending → Running → Degraded → Terminated`
- `thread.status`: `open → closed → archived`
- `hitl_callback.status`: `pending → responded → expired → escalated`

### Migrations

- Managed via `goose`
- Migration files in `db/migrations/` directory
- Applied via K8s Helm pre-install/pre-upgrade Job (production) or Makefile target (local)

## Acceptance Criteria

1. ER diagram matches implementation
2. All FK relationships enforced at DB level
3. State machine transitions enforced at domain level
4. Migrations are idempotent
