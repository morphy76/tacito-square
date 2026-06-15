# BUG-M6.8: Hardcoded Hub System Prompt and Lack of Template Parameterization

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M6.8                                                           |
| Status        | OPEN                                                               |
| Severity      | HIGH                                                               |
| Milestone     | M6 — Communities & Messaging                                       |
| Affects       | internal/agent/application/service/orchestrator.go, internal/keeper/adapters/outbound/crd/crd_coordinator.go, deploy/postgres/migrations |
| Violates      | SPEC-FR-M6.1, SPEC-FR-M3.4                                         |
| Discovered    | Code inspection of community orchestrator showing hardcoded hub prompt in Go code rather than template loaded in the database. |

## Problem Statement

The Hub agent's system prompt is currently hardcoded in the Go application logic within [orchestrator.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/application/service/orchestrator.go#L451-L464). This prevents operators and tenants from modifying or customizing the Hub's routing directives. 

Furthermore, prompt templates should be loaded into the database at initial migration time. These system-wide template entries must be protected against accidental modification or deletion (immutable/read-only) using negative identities (UUIDs representing negative integer structures). At runtime, the system prompt must act as a template with placeholders to dynamically compose the final version of the system prompt used by the Hub.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| Agent Orchestrator | [orchestrator.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/application/service/orchestrator.go) | The base system prompt and coordinate directives are hardcoded instead of being loaded from a parameterized template. |
| Keeper CRD Coordinator | [crd_coordinator.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/outbound/crd/crd_coordinator.go) | Synthesis logic does not support or load the system-seeded templates, and is not prepared to propagate template data to the agent. |
| Keeper Prompt Service & Repositories | [prompt_service.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/application/service/prompt_service.go), [prompt_repository.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/outbound/postgres/prompt_repository.go) | Lacks validation to prevent updates and deletions of system-locked templates (negative identities). |
| Database Migrations | [00001_init.sql](file:///Users/R.Pasquini/Projects/side/tacito-square/deploy/postgres/migrations/00001_init.sql) or new migration | Lacks initial seed data for the system-locked Hub prompt template. |

## Impact

1. **Lack of Customizability**: The orchestration behavior (JSON instructions, delegation rules) cannot be updated without a full application rebuild/deployment.
2. **Potential for Data Loss/Corruption**: System-level prompt templates could be modified or deleted via standard HTTP CRUD routes, breaking hub agents.

## Expected Behaviour

1. **System Seeding**: A system-locked Hub prompt template with placeholders (e.g. `{{.Description}}` and `{{.Spokes}}`) is seeded in the database at migration time under a designated negative UUID identity (`ffffffff-ffff-ffff-ffff-ffffffffffff`).
2. **Read-Only / Immutability**: Any attempts to update or delete prompt templates with negative identities (first 8 bytes set to `0xFF`) are rejected with a 400 Bad Request error.
3. **Multi-Tenant Sharing**: System templates seeded under `tenant_id = 'system'` can be resolved and retrieved by any tenant context.
4. **Runtime Rendering**: The Agent Orchestrator parses the system prompt as a Go template and executes it with dynamic parameters (agent description, community spokes list) to form the final system prompt.

## Acceptance Criteria

1. **Seeding Validation**: The `prompt_templates` database table contains the Hub system prompt template after running database migrations.
2. **Immutability Enforcement**: `PUT /api/v1/prompts/ffffffff-ffff-ffff-ffff-ffffffffffff` and `DELETE /api/v1/prompts/ffffffff-ffff-ffff-ffff-ffffffffffff` return a standard JSON error with `400 Bad Request`.
3. **Template Compilation**: Hub agent compiles system prompt dynamically replacing `{{.Description}}` and `{{.Spokes}}` variables.
4. **Unit & Integration Tests**: Added tests verify immutability checks, template compilation, and db seeding.
