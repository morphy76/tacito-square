# TASK-M6.6.1: Extend Event Schema and Payload for Handoff

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M6.6.1                                 |
| Status        | VERIFIED                                    |
| Spec          | SPEC-FR-M6.6                                |
| Depends On    | none                                        |

## Description

Extend the `AgentDelegationPayload` struct in the shared event package to support propagating conversation history turns (`ContextHistory`) to target Spokes.

## Work Items

1. **RED Phase**:
   - Write a unit test in `pkg/events/conversational_test.go` verifying that JSON serialization and deserialization of `AgentDelegationPayload` successfully encodes and decodes the new `ContextHistory` field populated with sample `ThreadTurn` items.
2. **GREEN Phase**:
   - Add the `ContextHistory []ThreadTurn` field to `AgentDelegationPayload` inside [conversational.go](file:///Users/R.Pasquini/Projects/side/tacito-square/pkg/events/conversational.go).
   - Ensure the field uses json tag `context_history,omitempty`.
   - Verify that the new unit test compiles and passes.
3. **REFACTOR Phase**:
   - Ensure styling and formatting match standard conventions.

## Acceptance Criteria

1. `AgentDelegationPayload` successfully serializes and deserializes the `context_history` field with zero data loss.
2. Unit tests pass successfully.
