# TASK-M2.2.1: Shared foundation test review and augmentation (RED)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M2.2.1                                 |
| Status        | TODO                                        |
| Spec          | SPEC-FR-M2.2                                |
| Phase         | RED                                         |
| Depends On    | none                                        |

## Description

Review existing tests in `internal/shared/` and write new tests for any missing spec requirements. Existing packages: config, health, observability, errors. Missing from spec: graceful shutdown helper. Tests must define the intended interface before implementation.

## Work Items

1. Review existing test coverage:
   - `internal/shared/config/config_test.go` — verify covers `Load()` and `LoadFromFile()`.
   - `internal/shared/health/health_test.go` — verify covers LivezHandler, ReadyzHandler, PingChecker, HTTPChecker.
   - `internal/shared/observability/logger_test.go` — verify covers NewLogger, WithTraceID, WithClaims.
   - `internal/shared/errors/errors_test.go` — verify covers sentinel errors, DomainError, Unwrap, constructors.
2. Write new tests for missing functionality:
   - `internal/shared/observability/tracing_test.go` — test InitTracer with empty endpoint (no-op), test Tracer() returns valid tracer.
   - `internal/shared/shutdown/shutdown_test.go` — test graceful shutdown helper: registers multiple cleanup functions, invokes them on signal, respects timeout.
3. Run `go test ./internal/shared/...` — new tests MUST FAIL for unimplemented code (shutdown package).

## Acceptance Criteria

1. All test files reviewed and augmented where needed.
2. `shutdown_test.go` exists and defines the graceful shutdown interface.
3. New shutdown tests FAIL (package doesn't exist yet).
4. Existing tests still PASS.
