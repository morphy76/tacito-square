# SPEC-FR-04.1: LLM Reasoning (Brain Adapter)

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-FR-04.1                       |
| Status        | VERIFIED                           |
| Milestone     | M1                                 |
| FR/NFR Ref    | FR-04.1                            |
| Component     | agent                              |
| Depends On    | —                                  |

## Context

Agents reason by sending conversation history to an OpenAI-compatible LLM API and receiving a response.

## Specification

1. The `Brain` outbound port MUST define `Reason(ctx, messages) → (Message, error)`.
2. The OpenAI adapter MUST call `POST /v1/chat/completions` with model, messages.
3. The adapter MUST include `Authorization: Bearer {apiKey}` header.
4. The adapter MUST return a domain `Message` with role `assistant`.
5. API errors (non-200) MUST return an error with status code.
6. Empty choices response MUST return an error.

## Acceptance Criteria

1. Successful reasoning returns assistant message with correct content
2. API 500 error returns descriptive error
3. Empty choices response returns "no choices" error
4. Request includes correct Authorization header and Content-Type

## Test Plan

- Unit tests with `httptest.NewServer` fixtures
- 3 test cases: success, API error, empty choices

## Files

- `internal/agent/adapters/outbound/openai/brain_adapter.go` ✅ IMPLEMENTED
- `internal/agent/adapters/outbound/openai/brain_adapter_test.go` ✅ 3 tests passing
