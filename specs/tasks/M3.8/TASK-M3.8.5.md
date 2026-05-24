# TASK-M3.8.5: Abstract Transaction Runner Port & Postgres Adapter Implementation

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M3.8.5                                 |
| Status        | IMPLEMENTED                                 |
| Spec          | SPEC-FR-M3.8                                |
| Depends On    | none                                        |

## Description

Design and implement a clean transaction abstraction inside the Keeper hexagonal application architecture. This eliminates the leakage of database-specific driver types (like pgx transactions) into application orchestrators, exposing a simple functional ports runner to execute multi-repository database actions inside a unified transactional boundary using context propagation.

## Boundary & Target Functions

- **Packages**:
  - `internal/keeper/application/ports/outbound`
  - `internal/keeper/adapters/outbound/postgres`
- **Files**:
  - `internal/keeper/application/ports/outbound/transaction.go` (Decoupled port definition)
  - `internal/keeper/adapters/outbound/postgres/transaction.go` (PostgreSQL adapter implementation)
  - `internal/keeper/adapters/outbound/postgres/agent_repository.go` (Repository updates)
- **Target Types & Functions**:
  - `ports.TransactionRunner` (Interface with `RunInTransaction`)
  - `postgres.TransactionRunner` (Implementation wrapping `pgxpool.Pool`)

## Work Items

1. **RED Phase**:
   - Write unit tests in `internal/keeper/adapters/outbound/postgres/transaction_test.go` using `pgxpool.Pool` to verify:
     - Basic operations succeed and commit correctly when running inside `RunInTransaction`.
     - Operations rollback automatically if the functional transaction runner callback returns an error.
     - Nested context gets injected with the transaction handle, which repository operations correctly utilize.

2. **GREEN Phase**:
   - Create `internal/keeper/application/ports/outbound/transaction.go` declaring the `TransactionRunner` interface.
   - Implement `internal/keeper/adapters/outbound/postgres/transaction.go` wrapping standard pgx pool transaction execution.
     - Leverage a custom context key to inject the active `pgx.Tx` transaction object into the returned context inside `RunInTransaction`.
   - Update `internal/keeper/adapters/outbound/postgres/agent_repository.go` and associated repositories:
     - Create a helper to extract the active transaction from the context (e.g. `tx, ok := txFromContext(ctx)`).
     - If an active transaction exists, use it to execute statements. If not, fallback to using the standard connection pool.

3. **REFACTOR Phase**:
   - Clean up explicit pgx `Begin` and `Commit` calls from repository adapter boundaries, replacing them with context-aware calls to the newly unified `TransactionRunner`.
   - Enforce clean import hierarchies ensuring pure packages inside `application/` remain free of `pgx` imports.

## Acceptance Criteria

1. All repository integration tests pass cleanly with transaction support.
2. An error in a transactional block causes an automatic SQL transaction rollback.
3. The domain and application use case structures remain completely decoupled from database client driver libraries.
