# SPEC-FR-10.1: Unit Tests (TDD)

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-FR-10.1                       |
| Status        | VERIFIED                           |
| Milestone     | M1+                                |
| FR/NFR Ref    | FR-10.1                            |
| Component     | all                                |
| Depends On    | —                                  |

## Context

Tacito Square follows strict TDD (Red → Green → Refactor). Every feature must have tests written before implementation.

## Specification

1. All domain logic, services, and adapters MUST have unit tests.
2. Tests MUST use `testify/assert` and `testify/require` for assertions.
3. Tests MUST use `testify/mock` or hand-rolled fakes for port dependencies.
4. Tests MUST be run with `-race` flag to detect data races.
5. `make test` MUST execute all unit tests with race detection.
6. Test coverage target: ≥80% for domain and service packages.

## Acceptance Criteria

1. `make test` passes with 0 failures
2. Race detector enabled
3. All packages under `internal/` have corresponding `_test.go` files
4. Mock/fake ports used (no concrete adapter dependencies in unit tests)

## M1 Test Inventory

| Package | Tests |
|---------|-------|
| `internal/agent/domain` | 11 |
| `internal/agent/service` | 3 |
| `internal/agent/adapters/outbound/openai` | 3 |
| `internal/agent/adapters/outbound/redis` | 7 |
| `internal/keeper/domain` | 11 |
| `internal/keeper/service` | 5 |
| `internal/keeper/adapters/inbound/httphandler` | 5 |
| `internal/shared/config` | 3 |
| `internal/shared/observability` | 6 |
| `internal/shared/errors` | 4 |
| `internal/shared/auth` | 7 |
| `internal/shared/health` | 7 |
| **Total** | **72** |

## Files

- All `*_test.go` files across `internal/` ✅
