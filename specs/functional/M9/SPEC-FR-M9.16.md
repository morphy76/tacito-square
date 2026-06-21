# SPEC-FR-M10.16: Provide configuration flags to opt-out of specific built-in tools

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M10.16                               |
| Status        | DRAFT                                       |
| Milestone     | M10                                          |
| Component     | keeper, agent                               |
| Depends On    | none                                        |
| Supersedes    | none                                        |

## Context

Allow agents to optionally disable specific default tools to enforce strict sandbox capabilities.

## Specification

1. Add configuration options to the Agent definition to specify excluded built-in tools.
2. During agent initialization, filter out any excluded default tools from the active tools list.
3. Validate agent execution to ensure disabled tools cannot be triggered.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
