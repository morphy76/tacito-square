# SPEC-NFR-LOG: Structured Logging

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-NFR-LOG                       |
| Status        | VERIFIED                           |
| Component     | keeper, agent, bff                 |

## Specification

1. All Golang components MUST use **zerolog** for structured JSON logging.
2. All typescript components MUST use **winston** for structured JSON logging.
3. Every log entry MUST include `trace_id` and `span_id` from the active OTel span context.
4. A configurable (at build time via `LogClaimsKeys` variable) set of JWT token claims MUST be included in log entries. Default: `["sub", "email"]`.
5. Log levels: `trace`, `debug`, `info`, `warn`, `error`. Default: `info`.
6. Output format: JSON to stdout. No text mode in production.

## Acceptance Criteria

1. `NewLogger("info", w)` produces JSON with `level`, `message`, `time` fields
2. `WithTraceID(logger, spanCtx)` adds `trace_id` and `span_id` fields
3. `WithClaims(logger, claims)` adds only keys listed in `LogClaimsKeys`
4. Debug messages are suppressed at info level
