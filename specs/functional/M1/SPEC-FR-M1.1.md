# SPEC-FR-M1.1: Build System & Layout

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M1.1                                |
| Status        | VERIFIED                                    |
| Milestone     | M1                                          |
| FR/NFR Ref    | FR-M1.1                                     |
| Component     | shared                                      |
| Depends On    | none                                        |
| Supersedes    | none                                        |

## Context
Tacito Square requires a robust initial project structure and build system to enable the rapid development of the components. This spec covers the basic layout and the makefile targets.

## Specification
The system MUST include:
1. A standard Go project directory structure (e.g., `cmd/`, `internal/`, `pkg/`, `scripts/`).
2. A `Makefile` providing commands to build the project and run tests.

## Acceptance Criteria
1. `make build` successfully builds all binaries.
2. `make test` executes independent tasks for unit, integration (using testcontainers), benchmark, and race tests.

## Test Plan
- Run `make build` and verify binaries are produced.
- Run `make test` to verify all test suites are executed.

## API Contract (if applicable)
N/A

## Files Affected
- `Makefile`
- `cmd/*`
- `internal/*`
- `pkg/*`
