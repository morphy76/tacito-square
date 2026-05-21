# SPEC-FR-M5.2: LLM Reasoning (Brain Adapter)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M5.2                                |
| Status        | DRAFT                                       |
| Milestone     | M5                                          |
| Component     | agent                                       |
| Depends On    | SPEC-FR-M5.1                                |
| Supersedes    | none                                        |

## Context

Agents reason by sending prompts to an LLM and processing responses. The brain adapter is an outbound port implementation that abstracts the LLM provider, enabling provider-agnostic reasoning.

## Specification

1. The system MUST define a `Brain` outbound port in the agent domain layer.
2. The system MUST implement an OpenAI-compatible adapter using the `openai-go` SDK (per SPEC-NFR-STACK).
3. The adapter MUST support configurable model, temperature, and max tokens.
4. The adapter MUST support streaming responses.
5. The adapter MUST implement retry with exponential backoff and jitter on transient failures (per SPEC-NFR-CLOUD).
6. The adapter MUST propagate OpenTelemetry context for distributed tracing (per SPEC-NFR-REACTIVE).

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
