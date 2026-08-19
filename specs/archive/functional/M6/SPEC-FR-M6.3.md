# SPEC-FR-M6.3: NATS Subject Namespacing

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M6.3                                |
| Status        | VERIFIED                                    |
| Milestone     | M6                                          |
| Component     | agent, keeper                               |
| Depends On    | SPEC-FR-M6.2                                |
| Supersedes    | none                                        |

## Context

NATS subjects must be namespaced to isolate communities and prevent unauthorized cross-community communication.

## Resolution

This specification has been fully implemented and verified under:
- **[SPEC-FR-M6.0](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/functional/M6/SPEC-FR-M6.0.md)**: Establishes event-driven foundation, generic domain event schemas, and wildcard SSE stream subscriptions.
- **[SPEC-FR-M6.1](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/functional/M6/SPEC-FR-M6.1.md)**: Establishes community topology routing (Centralized Hub vs Single Agent queue routing).
- **[BUG-M6.3](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/tasks/M6.BUG3/BUG-M6.3.md)**: Fixes inbound routing subjects for coordinated communities (`ts.community.{id}.agent.hub` vs. `ts.community.{id}.agent.all` fallbacks).

## Specification (Historical)

1. NATS subjects MUST follow the format `ts.community.{community_id}.agent.{agent_id}`.
2. Community broadcast MUST use wildcard `ts.community.{community_id}.agent.*`.
3. Keeper monitoring subjects MUST use `ts.keeper.{agent_id}` for housekeeping.
4. Agents MUST only subscribe to subjects within their assigned community.
5. Subject authorization SHOULD be enforced via NATS account configuration.

