# TASK-M6.6.3: Implement Spoke-Side Memory Sync

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M6.6.3                                 |
| Status        | VERIFIED                                    |
| Spec          | SPEC-FR-M6.6                                |
| Depends On    | TASK-M6.6.1                                 |

## Description

Implement memory synchronization on the Spoke side. When a Spoke receives an `AgentDelegationPayload` carrying a non-empty `ContextHistory` list:
1. Clear the Spoke's private Short-Term Memory (STM) for the thread.
2. Repopulate the Spoke's private STM with the turns received in `ContextHistory` sequentially.
3. Delegate message processing to the reasoning engine processor.

## Work Items

1. **RED Phase**:
   - Write a unit test in [schema_router_impl_test.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/application/service/schema_router_impl_test.go) verifying that when a Spoke receives `AgentDelegationPayload` with sample history turns:
     - The private STM of the Spoke is populated with these turns in the exact order.
     - The message processor is called with the concatenated delegation message.
2. **GREEN Phase**:
   - Modify `handleAgentDelegation` in [schema_router_impl.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/application/service/schema_router_impl.go) to clear the STM and loop-append the `ContextHistory` turns prior to calling `processor.ProcessIncomingMessage`.
   - Verify that the unit test compiles and passes.
3. **REFACTOR Phase**:
   - Ensure clean timestamp parsing (handling RFC3339 format, fallback to current time on parse error).

## Acceptance Criteria

1. Target Spoke's local isolated STM is completely overwritten/synchronized with the Hub's global `ContextHistory` on delegation events.
2. Standard flow functionality (cognitive loop reasoning) completes successfully on top of the populated STM.
