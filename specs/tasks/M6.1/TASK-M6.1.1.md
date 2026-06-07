# TASK-M6.1.1: Keeper Database Schema & API Validation

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M6.1.1                                 |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M6.1                                |
| Depends On    | none                                        |

## Description

Introduce the `topology` field to the `communities` schema and the `role` field to the `agents` schema. Implement model and API validations to enforce the constraints of single-agent and hub-spoke topologies, as well as topology mutability limits.

## Boundary & Target Functions

- **Package**: `internal/keeper/domain/model`, `internal/keeper/adapters/inbound/http`, and migrations.
- **Files**:
  - `internal/keeper/domain/model/community.go`
  - `internal/keeper/domain/model/agent.go`
  - `internal/keeper/adapters/inbound/http/community_handler.go`
  - `internal/keeper/adapters/inbound/http/agent_handler.go`
  - DB Migration file (e.g. `internal/keeper/adapters/outbound/db/migrations/xxxx_add_topology_and_role.sql` or similar goose migration path)

## Work Items

1. **RED Phase (Write Tests First)**:
   * Write unit tests for community and agent model validation:
     * Assert validation fails if attempting to add a second agent to a `single-agent` community.
     * Assert validation fails if attempting to transition a `hub-spoke` community to active/deployed state without exactly one hub agent assigned.
     * Assert validation fails if attempting to update a community's topology type when it has one or more agents assigned.
   * Write handler HTTP integration tests for Keeper API:
     * Verify `PATCH /api/v1/communities/:id` returns `400 Bad Request` or a validation error when attempting to change topology of a non-empty community.

2. **GREEN Phase (Implement Minimum Code)**:
   * Create database migrations adding:
     * `communities.topology` column (VARCHAR, default `single-agent`).
     * `agents.role` column (VARCHAR, default `spoke`).
   * Update domain models to add `Topology` and `Role` fields.
   * Implement validation functions in models and intercept updates in the handler layer.
   * Enforce constraint rules upon agent assignment and community status transitions.

3. **REFACTOR Phase (Clean & Generalize)**:
   * Ensure database transaction safety when validating and updating assignments.
   * Standardize error formats returning JSON `{"error": "descriptive message"}` according to HTTP conventions.

## Acceptance Criteria

1. Database migrations run successfully.
2. Keeper REST APIs reject topology changes on communities that already have agents.
3. Keeper REST APIs prevent adding multiple agents to `single-agent` communities.
4. Keeper REST APIs prevent deploying `hub-spoke` communities without exactly one hub agent.
