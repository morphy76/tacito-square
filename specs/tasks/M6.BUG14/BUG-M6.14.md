# BUG-M6.14: Handoff Target Not Found Missing Observability

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M6.14                                                          |
| Status        | CLOSED                                                             |
| Severity      | LOW                                                                |
| Milestone     | M6 — Communities & Messaging                                       |
| Affects       | [orchestrator.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/application/service/orchestrator.go) |
| Violates      | `observability` / `SPEC-NFR-OBSERVABILITY`                         |
| Discovered    | Code audit of the handoff delegation flow.                         |

## Problem Statement

When a Spoke agent suggests a handoff delegation, the Orchestrator validates the target agent against the community's registered agent cards (lines 205-217 in `ProcessSpokeResponse`). If the target agent is not found (i.e. `targetCard` remains `nil`), the Orchestrator silently falls through to the normal execution flow (line 305):
```go
	if targetCard != nil {
		// Handoff Execution
        ...
		return nil
	}

	// Normal Flow / Fallback when handoff is target-invalid or not requested
	spokeEntry := model.MemoryEntry{
        ...
```
There is no warning log, trace event, or metric recorded to indicate that a handoff was requested but failed because the destination agent did not exist in the community. This lack of observability makes troubleshooting failed handoff suggestions in production environments extremely difficult.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| Orchestrator Service | [orchestrator.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/application/service/orchestrator.go) | Missing warning logging or metrics capture when handoff target validation fails. |

## Impact

1. **Undetected Failures:** Dynamic community routing decisions that fail due to missing configuration or typos in target names are masked as normal spoke replies.
2. **Operations Overhead:** Operations teams have no trace or log records to identify why expected agent handoffs did not trigger, violating observability rules.

## Expected Behaviour

1. **Warning Logging:** The Orchestrator must log a structured warning using `zerolog` if `isHandoff` is true but `targetCard` is `nil`, explicitly naming the missing target agent and the suggesting spoke.
2. **Metrics Collection:** Increments a failed handoff counter metric (or tags a routing error metric) indicating target validation failure.
3. **Trace Exception:** Records an exception or attribute on the active OpenTelemetry span signifying the invalid handoff target.

## Acceptance Criteria

1. A structured warning log is emitted when a handoff target is not found in the community cards directory.
2. A Prometheus counter metric is updated or an OTel attribute/exception is recorded on the span to capture the failure.
3. Unit tests verify that invalid handoff requests result in warning logs and appropriate metric/trace side-effects.
