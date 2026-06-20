# SPEC-FR-M6.6: Coordinated Conversation Handoff

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M6.6                                |
| Status        | DRAFT                                       |
| Milestone     | M6                                          |
| Component     | agent                                       |
| Depends On    | SPEC-FR-M6.1, SPEC-FR-M5.3, SPEC-FR-M6.0    |
| Supersedes    | none                                        |

## Context

During a conversational reasoning loop, a Spoke agent may determine that another agent within the community is better suited to answer the user's request. Handoff facilitates transferring the conversational task to the target agent.

To keep the architecture simple, clean, and secure:
1. **Coordinated Hub-Spoke Only**: Decentralized P2P handoffs are postponed to Milestone 11. The Hub orchestrator remains the sole controller of the community execution loop.
2. **Isolated Keyspaces**: Short-Term Memory (STM) keyspaces remain strictly agent-isolated to avoid data clutter and race conditions from concurrent spoke execution. The Hub explicitly propagates necessary context.
3. **Structured Suggestions**: Spokes propose handoffs by returning a structured JSON response to the Hub.

## Specification

### 1. Spoke-Initiated Handoff Suggestion (Option A)

If a Spoke agent determines it cannot fulfill a task and another agent is better suited, it MUST output a structured JSON response instead of a normal text message:

```json
{
  "action": "suggest_handoff",
  "target": "translator",
  "reason": "The message is in Spanish and needs translation before processing."
}
```

* This JSON string is set as the `Response` payload in the Spoke's `AgentResponsePayload` on NATS subject:
  `ts.community.{community_id}.agent.{spoke_id}.thread.{thread_id}.response`

### 2. Hub-Side Orchestration and Context Propagation

When the Hub orchestrator processes a Spoke's response:
1. **Handoff Detection**: The Hub attempts to parse the response as a handoff suggestion JSON. If detected:
   * The Hub validates if the target Spoke (e.g. `translator`) exists in the community registry (via Agent Cards discovery).
   * If the target is invalid/missing, the Hub falls back to its normal reasoning loop to handle the failure.
2. **Context Extension**: Since STM is agent-isolated, the target Spoke has no visibility into the thread's prior history. To solve this, the `AgentDelegationPayload` is extended to include the context history:
   ```go
   type AgentDelegationPayload struct {
       ThreadID        string       `json:"thread_id"`
       CommunityID     string       `json:"community_id"`
       DelegatingAgent string       `json:"delegating_agent"`
       TargetAgent     string       `json:"target_agent"`
       Message         string       `json:"message"`
       ContextHistory  []ThreadTurn `json:"context_history,omitempty"` // Explicit history window for the Spoke
   }
   ```
3. **Handoff Execution**: The Hub updates its state machine in Redis, populates `ContextHistory` with the thread's recent turns, and publishes the new delegation task to the target Spoke.

---

## Technical Debt: Hardcodings Tracking

To support future extensibility and allow alternative community configurations, the following hardcodings within this process are tracked for future removal:

1. **System Prompt Directives**: The Hub's default system prompt template and routing instructions (`delegate`, `finalize`, and the new `suggest_handoff` schemas) are hardcoded inside [orchestrator.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/application/service/orchestrator.go).
2. **Action Code Constraints**: Parsing of JSON actions (`"delegate"`, `"finalize"`, `"suggest_handoff"`) relies on hardcoded string literals and conditional branching inside the Go orchestration services.
3. **Redis Keyspace Format**: The Redis keyspace format `ts:tenant:%s:agent:%s:stm:%s` is hardcoded inside [memory_adapter.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/adapters/outbound/redis/memory_adapter.go).
4. **Hardcoded Brain Prompts**: Prompt templates for text polishing ([schema_router_impl.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/application/service/schema_router_impl.go)) and memory consolidation ([message_processor.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/application/service/message_processor.go)) are hardcoded in string constants.
5. **NATS Subject Templates**: Subject structure formats (e.g., `ts.community.%s.agent.hub`) are formatted using hardcoded string structures in adapters.

---

## Acceptance Criteria

1. The Spoke agent can output `suggest_handoff` JSON when it lacks the capability to complete a task.
2. The Hub orchestrator correctly parses `suggest_handoff` from Spoke responses.
3. The Hub validates the handoff target against active Agent Cards.
4. The Hub propagates recent thread history turns in `ContextHistory` when delegating to the target Spoke.
5. The target Spoke successfully starts its reasoning using the provided `ContextHistory`.

## Test Plan

* **Unit Tests**:
  * Verify parsing of `suggest_handoff` payloads on the Hub.
  * Verify validation logic against missing/invalid target agents.
  * Verify that the Hub correctly populates `ContextHistory` in `AgentDelegationPayload`.
* **Integration Tests**:
  * Set up a Hub and two Spokes (`writer`, `translator`).
  * Send a message requiring translation to `writer`.
  * Verify `writer` suggests handoff to `translator`.
  * Verify `translator` receives the delegation with `ContextHistory` and successfully responds.
