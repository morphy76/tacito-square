# BUG-M6.11: Spoke Response CoT Leakage due to Missing Polishing on Delegated turns

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M6.11                                                          |
| Status        | CLOSED                                                             |
| Severity      | MEDIUM                                                             |
| Milestone     | M6 — Communities & Messaging                                       |
| Affects       | `internal/agent/application/service/schema_router_impl.go`          |
| Violates      | SPEC-FR-M6.6                                                       |
| Discovered    | Spoke response payloads in thread history containing raw reasoning monologues and thought preambles. |

## Problem Statement

When Spoke agents processed delegation messages, they generated responses containing raw internal reasoning and Chain-of-Thought (CoT) monologues (e.g., *"The user wants to learn... my role is to... Therefore, a good next question would be..."*). 

This occurred because in `schema_router_impl.go`, the `handleAgentDelegation` function (which handles NATS agent-delegation events) omitted the `ensureHumanReadable` polishing step. This polishing step was only present in `handleAddUserMessage` (which handles user-facing requests), leaving the delegated spoke response raw and unpolished.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| Schema Router Implementation | `internal/agent/application/service/schema_router_impl.go` | Omitted the `ensureHumanReadable` polishing call in the `handleAgentDelegation` event handler. |

## Impact

1. Raw agent monologue, reasoning steps, and planning details leaked directly into the chat history.
2. Contaminated the Hub's thread context with unstructured reasoning data, affecting subsequent routing turns.

## Expected Behaviour

1. All Spoke responses (whether triggered by direct user messages or delegated tasks from the Hub) MUST be polished using the brain's `ensureHumanReadable` helper before being emitted back to the community.

## Acceptance Criteria

1. `handleAgentDelegation` calls `ensureHumanReadable` on the processed Spoke response before constructing the response payload.
2. Unit tests verify that polishing is applied successfully during agent delegation.
