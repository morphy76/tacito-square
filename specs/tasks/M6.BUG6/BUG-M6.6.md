# BUG-M6.6: Orchestrator Loop Limit Terminating Threads Instead of Returning Latest Spoke Response

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M6.6                                                           |
| Status        | CLOSED                                                             |
| Severity      | HIGH                                                               |
| Milestone     | M6 — Communities & Messaging                                       |
| Affects       | `internal/agent/application/service/orchestrator.go`               |
| Violates      | SPEC-FR-M6.1                                                       |

## Problem Statement

When a community Hub agent coordinates a conversation with multiple spoke agents, it can enter a loop (for example, when a Spoke asks a clarifying question to the user and the other Spoke acknowledges it, but the Hub decides to delegate again instead of yielding to the user). When this loop limit is reached (currently set dynamically at flow start), the Hub aborts the entire thread:
1. It sends an `EndThread` event to all Spokes, wiping their short-term memories.
2. It clears the orchestration state.
3. It returns a generic fallback error message: "Orchestration limit exceeded without reaching a final answer."

This results in a terminated conversation and lost context.

## Expected Behaviour

1. Reaching the loop limit should not terminate the thread (do not send `EndThread`).
2. The latest response received from the Spokes should be returned as the final message to the user, allowing the conversation to continue when the user replies.
3. The system prompt should include directives to finalize when a Spoke agent asks a clarifying question or indicates missing information.
