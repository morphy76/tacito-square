# BUG-M6.13: Missing Self-Delegation Runtime Guard

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M6.13                                                          |
| Status        | OPEN                                                               |
| Severity      | MEDIUM                                                             |
| Milestone     | M6 — Communities & Messaging                                       |
| Affects       | [orchestrator.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/application/service/orchestrator.go) |
| Violates      | `SPEC-FR-M6.1` / `SPEC-FR-M6.6`                                    |
| Discovered    | Code audit of the orchestration delegation flow.                   |

## Problem Statement

The orchestrator lacks robust guards against self-delegation (delegating tasks to itself) and case-sensitive checks for its own final response:

1. **Missing Runtime Guard on Delegation:** While `compileSystemPrompt` filters the Hub's own card name (lines 612-614) to prevent it from appearing in the prompt as a candidate Spoke, the runtime delegation parsing loop (lines 459-475 in `runOrchestrationTurn`) does not perform any validation. If the LLM hallucinations return the Hub's own name in the delegation list, the system will proceed to queue a NATS task event targeting itself.
2. **Case-Sensitive Final Response Guard:** In `ProcessSpokeResponse` (line 141), the guard to ignore its own final response uses a simple case-sensitive comparison:
   ```go
   if payload.AgentName == o.agentName {
   ```
   If the casing varies due to environment configurations or event metadata differences, this check fails, causing the Hub to process its own final response as a Spoke response.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| Orchestrator Service | [orchestrator.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/application/service/orchestrator.go) | Missing runtime self-delegation validation in `runOrchestrationTurn` and case-sensitive check in `ProcessSpokeResponse`. |

## Impact

1. **Orchestration Loops / Infinite Execution:** If the Hub delegates a task to itself, it will trigger an infinite self-delegation loop, consuming execution credits, memory resources, and publishing redundant NATS messages.
2. **State Pollution:** Processing its own final response as a Spoke response contaminates the orchestration state and disrupts the conversation state machine.

## Expected Behaviour

1. **Runtime Validation:** In `runOrchestrationTurn`, if the list of target spokes contains the Hub's own name (case-insensitive), it should be filtered out, logged, and ignored (or raise an error if it's the only target).
2. **Case-Insensitive Guards:** All checks comparing incoming agent names to the Hub's own name (`o.agentName`) MUST use case-insensitive matching (e.g., `strings.EqualFold`).

## Acceptance Criteria

1. Runtime check in `runOrchestrationTurn` filters out self-delegation attempts.
2. The self-response guard in `ProcessSpokeResponse` uses case-insensitive matching (`strings.EqualFold`).
3. Unit tests verify that self-delegation is rejected or safely filtered out.
4. Unit tests verify that variations in casing for the Hub's name are handled correctly by the final response guard.
