# SPEC-FR-M6.5: A2A Agent Cards

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M6.5                                |
| Status        | DRAFT                                       |
| Milestone     | M6                                          |
| Component     | agent                                       |
| Depends On    | SPEC-FR-M5.1                                |
| Supersedes    | none                                        |

## Context

Each agent publishes an Agent Card describing its capabilities, per the A2A protocol. Agent Cards enable discovery and capability-based routing within communities.

## Specification

1. Each agent MUST generate an Agent Card on startup from its configuration.
2. The Agent Card MUST include: agent name, description, capabilities, supported message types, community memberships.
3. The Agent Card MUST be published to NATS on agent startup and on configuration changes.
4. The Agent Card MUST be retrievable at `GET /.well-known/agent.json`.
5. Hub agents MUST maintain a registry of spoke Agent Cards for routing decisions.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
