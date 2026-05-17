# SPEC-NFR-LOG: Structured Logging

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-NFR-LOG                       |
| Status        | VERIFIED                           |
| Milestone     | M1                                 |
| Component     | shared                             |

## Specification

1. All components MUST use **zerolog** for structured JSON logging.
2. Every log entry MUST include `trace_id` and `span_id` from the active OTel span context.
3. A configurable (at build time via `LogClaimsKeys` variable) set of JWT token claims MUST be included in log entries. Default: `["sub", "email"]`.
4. Log levels: `trace`, `debug`, `info`, `warn`, `error`. Default: `info`.
5. Output format: JSON to stdout. No text mode in production.

## Acceptance Criteria

1. `NewLogger("info", w)` produces JSON with `level`, `message`, `time` fields
2. `WithTraceID(logger, spanCtx)` adds `trace_id` and `span_id` fields
3. `WithClaims(logger, claims)` adds only keys listed in `LogClaimsKeys`
4. Debug messages are suppressed at info level

## Files

- `internal/shared/observability/logger.go` ✅ IMPLEMENTED
- `internal/shared/observability/logger_test.go` ✅ 6 tests passing
