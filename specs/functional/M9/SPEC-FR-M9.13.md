# SPEC-FR-M10.13: Track brain token usage per agent and thread

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M10.13                               |
| Status        | DRAFT                                       |
| Milestone     | M10                                          |
| Component     | agent, keeper                               |
| Depends On    | none                                        |
| Supersedes    | none                                        |

## Context

Track and persist input and output LLM token counts consumed by agent reasoning cycles, aggregated per thread and per agent definition.

## Specification

1. Intercept brain LLM responses to extract token usage counts (input tokens, output tokens).
2. Propagate token usage reports back to the Keeper database.
3. Expose aggregated metrics for billing and resource monitoring.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
