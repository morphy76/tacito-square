# BUG-M3.12: Agent Definition Lacks Strict Enforcement of Brain Requirement

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M3.12                                                          |
| Status        | OPEN                                                               |
| Severity      | HIGH                                                               |
| Milestone     | M3 — Keeper Core                                                   |
| Affects       | internal/keeper/domain/agent.go, internal/keeper/adapters/http/agent_handlers.go |
| Violates      | SPEC-FR-M3.5 §1 (Agent Domain Model)                               |
| Discovered    | Architectural compliance review of Agent validation invariants      |

## Problem Statement

The system fails to enforce that an **Agent** must have a functional, validly configured **Brain** under all lifecycle phases. Although the domain model checks that `a.Brain.Model != ""` and some basic constraints are verified, the HTTP payload parser and persistence layers do not sufficiently prevent the saving or transitions of agents that lack a fully populated brain (e.g. empty endpoints or missing credentials secret), or allow uninitialized or empty brain configurations during GORM mapping or API requests.

An agent is a computational unit designed to process messages asynchronously, which fundamentally requires a Brain (an LLM adapter utilizing OpenAI REST API signatures) to operate. Allowing an agent template to be saved or assigned to communities without a strictly validated brain causes runtime failures in the reconciliation controller (M4) and agent core (M5) during container startup and execution.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| `Agent` Domain Model | `internal/keeper/domain/agent.go` | Validation invariants only check the model string and temperature, but do not enforce that `Endpoint` and `CredentialsSecret` are populated for active or assigned agents. |
| `http` Handlers | `internal/keeper/adapters/http/agent_handlers.go` | `CreateAgentRequest` and `UpdateAgentRequest` validate the `brain` object via gin bindings, but fail to reject incomplete brain blocks when updating or creating templates. |
| DB Schema & GORM | `deploy/postgres/migrations/00005_create_agents_table.sql` | The database accepts any `JSONB` value for the `brain` column without schema validation or constraints enforcing non-empty fields. |

## Impact

1. **Reconciliation Failures**: When the Operator (M4) attempts to provision a Kubernetes deployment for an agent, it extracts brain configuration secrets. If these are empty or invalid, the container fails to start, leading to a `CrashLoopBackOff` or unhandled deployment errors in the operator.
2. **Runtime Exceptions**: The Agent container (M5) crashes or fails to process incoming NATS messages when invoking LLM reasoning due to nil or blank provider details.
3. **Data Integrity Issues**: Allows the database to accumulate corrupt or useless agent templates that cannot ever be successfully deployed.

## Expected Behaviour

1. Every saved or assigned agent MUST have a fully-configured `Brain` template.
2. The domain `Validate()` method must perform strict validation on `BrainConfig`:
   - `Model` must be non-empty and match supported LLM providers.
   - `Endpoint` must be a valid URL if specified, or a non-empty string.
   - `CredentialsSecret` must be non-empty.
3. The HTTP API must return a clear validation error (`400 Bad Request`) using standard JSON error schemas when these validation rules are violated.

## Acceptance Criteria

1. Creation (`POST`) or Update (`PUT`) of an Agent template fails with `400 Bad Request` if `brain.model`, `brain.endpoint`, or `brain.credentials_secret` is blank or invalid.
2. Domain validation unit tests assert that an Agent without a fully defined Brain is rejected with appropriate domain error messages.
3. The database migration schema or repository layer enforces that saved brain configurations are structurally valid.
