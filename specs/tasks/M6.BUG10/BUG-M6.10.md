# BUG-M6.10: Hub-Spoke Orchestration Observation Role Confusion and Loop

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M6.10                                                          |
| Status        | CLOSED                                                             |
| Severity      | HIGH                                                               |
| Milestone     | M6 — Communities & Messaging                                       |
| Affects       | `internal/agent/application/service/orchestrator.go`, `internal/agent/adapters/outbound/openai/openai_adapter.go`, `internal/agent/adapters/outbound/ollama/ollama_adapter.go` |
| Violates      | SPEC-FR-M6.1                                                       |
| Discovered    | Thread history logs showing infinite delegation loops and MaxLoops safeguard trigger. |

## Problem Statement

The Hub orchestrator would enter an infinite delegation loop with Spoke agents (e.g. `'answerer'` and `'enquirer'`) when they returned clarifying questions, eventually terminating only when the loop safeguard count exceeded `MaxLoops`. 

This was caused by two main factors:
1. **Identity Bleed / Role Confusion:** The Hub logged Spoke response observations in short-term memory (STM) using `"role": "user"`. Furthermore, outbound LLM adapters (`openai` and `ollama`) fell back to formatting any unknown history role (including `"system"`) as `"user"` message turns. Consequently, the orchestrator LLM saw system observations and user queries under the same `"user"` role, leading to confusion and repeated delegation.
2. **History Splitting Regression:** In `runOrchestrationTurn`, the latest history entry (the latest Spoke observation) was sliced off and passed as the primary prompt. Because the brain adapters formatted this prompt as a user message, the observation was treated as a user turn, causing the LLM to recycle it.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| Orchestrator Service | `internal/agent/application/service/orchestrator.go` | Appended observations as `"role": "user"` and sliced the latest observation as the primary prompt. |
| OpenAI Brain Adapter | `internal/agent/adapters/outbound/openai/openai_adapter.go` | Mapped `"system"` role history turns to `"user"` API messages. |
| Ollama Brain Adapter | `internal/agent/adapters/outbound/ollama/ollama_adapter.go` | Mapped `"system"` role history turns to `"user"` API messages. |

## Impact

1. Conversations with clarifying questions degenerated into infinite loops, consuming unnecessary LLM tokens.
2. Thread was force-closed at loop limit with a raw state/fallback response instead of a cohesive final answer.

## Expected Behaviour

1. Spoke observations MUST be logged as `"role": "system"` (or system turns) to distinguish them from the user's queries.
2. Outbound LLM adapters MUST map the `"system"` role turns to system messages.
3. The Hub MUST pass the entire conversation history as history context and use a static coordination prompt to coordinate next turns.

## Acceptance Criteria

1. Spoke observations are logged as `"system"` role turns in memory.
2. Brain requests contain system-mapped history and a static coordination prompt.
3. Unit tests verify the new orchestration loop and prompt layout.
