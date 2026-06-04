# SPEC-FR-M5.1: Agent Configuration from CRD Spec

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M5.1                                |
| Status        | VERIFIED                                    |
| Milestone     | M5                                          |
| Component     | agent                                       |
| Depends On    | SPEC-FR-M4.1                                |
| Supersedes    | none                                        |

## Context

When an agent pod starts, it must load its configuration from environment variables derived from the TacitoAgent CRD spec. This includes LLM settings, memory configuration, tool bindings, and community context.

## Specification

1. The agent MUST load configuration from environment variables using Viper (per SPEC-NFR-STACK).
2. Environment variables MUST follow the naming convention `TS_AGENT_*`.
3. Required configuration: LLM model, LLM endpoint, community ID, agent ID, NATS URL, Redis URL, Qdrant URL.
4. Optional configuration: temperature, max tokens, system prompt (inline or reference), S3 endpoint.
5. The agent MUST fail fast on missing required configuration with a clear error message.
6. The agent MUST log its resolved configuration at startup (per SPEC-NFR-LOG), redacting sensitive values.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
