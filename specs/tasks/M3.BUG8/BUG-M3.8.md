# BUG-M3.8: Stack Dependencies & Migration Framework Deviations

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M3.8                                                           |
| Status        | IMPLEMENTED                                                        |
| Severity      | MEDIUM                                                             |
| Milestone     | M3 — Keeper Core                                                   |
| Affects       | go.mod, internal/keeper/migrations.go                              |
| Violates      | SPEC-NFR-STACK §24, §25                                            |
| Discovered    | M3 candidate post-implementation NFR review                         |

## Problem Statement

The tech stack definitions defined in `SPEC-NFR-STACK` are strictly locked, requiring constitutional amendments to alter. However, the current M3 candidate implementation contains deviations:

1. **Custom Migration Engine instead of Goose**: The spec locks `goose` as the migration framework. Rather than importing and calling the `goose` Go library programmatically in `internal/keeper/migrations.go`, the application implements a custom, raw SQL parser and executor using pgx connections:
   ```go
   // Parse the UP migration SQL block (before "-- +goose Down" comment)
   var upSql string
   downIndex := strings.Index(content, "-- +goose Down")
   // ...
   // Clean up goose directive lines
   upSql = strings.ReplaceAll(upSql, "-- +goose Up", "")
   ```
   Implementing custom, proprietary migration parsing logic is fragile and deviates from the locked technology stack.
2. **Missing go.mod Package Declarations**: The locked packages (including `github.com/redis/go-redis`, `github.com/nats-io/nats.go`, `github.com/modelcontextprotocol/go-sdk`, `github.com/zitadel/oidc/v3`, etc.) are completely absent from `go.mod` because they are not yet integrated into the M3 candidate.

## Affected Aggregates and Files

| File / Component | Location | Issue |
|------------------|----------|-------|
| `migrations.go` | `internal/keeper/migrations.go` | Implements custom parsing logic for migration files instead of executing via the `goose` engine. |
| `go.mod` | `go.mod` | Missing required locked stack packages. |

## Impact

1. **Fragile Database Deployments**: Custom SQL string manipulation to extract migrations is error-prone and lacks transaction management, state tracking (e.g., `goose_db_version` schema table), or fallback capabilities built into mature engines like goose.
2. **Delayed Stack Validation**: Delaying package imports in `go.mod` bypasses compiler and dependency trees validation at the early Milestone stage.
3. **Compliance Failure**: Violates SPEC-NFR-STACK requirements.

## Expected Behaviour

1. Keeper database migrations MUST be applied programmatically using the official `goose` library or the goose CLI command runner.
2. Core locked stack dependencies MUST be declared in `go.mod` to align dependencies.

## Acceptance Criteria

1. Database schema migration runs programmatically via the approved `goose` driver.
2. `go.mod` lists only approved dependencies and includes all locked packages necessary for M3 and immediate downstream milestones (M4/M5).
