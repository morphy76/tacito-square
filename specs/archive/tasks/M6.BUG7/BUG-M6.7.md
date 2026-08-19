# BUG-M6.7: Agent Messages Pollution and Lack of Classification in Server-Sent Events (SSE) Stream

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M6.7                                                           |
| Status        | CLOSED                                                             |
| Severity      | HIGH                                                               |
| Milestone     | M6 — Communities & Messaging                                       |
| Affects       | `internal/keeper/adapters/inbound/http/event_handlers.go`, `pkg/events/conversational.go`, `internal/agent/application/service/orchestrator.go`, `internal/agent/application/service/schema_router_impl.go` |
| Violates      | SPEC-FR-M6.0, SPEC-FR-M6.1                                         |
| Discovered    | SSE monitoring stream inspection when using Hub-Spoke communities |

## Problem Statement

When using a Hub-Spoke community, the Server-Sent Events (SSE) stream (which acts as a user-facing monitoring channel) is polluted and unclear. Specifically:

1. **Pollution of `add-user-message` events**: When the Hub agent delegates tasks to Spoke agents, it publishes `add-user-message` events with a source indicating they came from the agent (e.g., `agent/<agent-id>`). This violates the semantics of `add-user-message`, which should only represent messages *sent by the user*.
2. **Lack of Event Classification**: All agent response events (Hub progression updates, Spoke responses, the final Hub response, and standalone responses) are published under the same schema `urn:tacito:schema:conversational:agent-response:v1` with no distinction in the payload. The client cannot programmatically distinguish what is intermediate reasoning, what is a spoke response, or what is the final answer.
3. **Messy Concatenated Final Answer**: Because clients cannot classify the stream events, they concatenate all incoming agent-response events together, resulting in a single text bubble containing the Hub's coordination logs, Spoke responses, and final response. Additionally, storing Spoke responses as `"assistant"` turns in the Hub's STM confuses the LLM (which sees them as its own past outputs), and the Hub's system prompt lacks instructions to synthesize these inputs cleanly.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| Keeper HTTP Inbound | `internal/keeper/adapters/inbound/http/event_handlers.go` | Streams all events, but needs to handle the distinct segment extraction for `agent-delegation`. |
| Event Schema | `pkg/events/conversational.go` | `AgentResponsePayload` does not include classification metadata. Missing a dedicated inter-agent delegation event schema. |
| Agent Orchestrator | `internal/agent/application/service/orchestrator.go` | Publishes `add-user-message` for delegation, does not populate classification in responses, and appends Spoke responses to STM as `"assistant"`. |
| Agent Router | `internal/agent/application/service/schema_router_impl.go` | Does not handle dedicated inter-agent delegation events or set standalone response classifications. |

## Impact

1. UIs or clients monitoring the SSE channel cannot render a clean conversation feed, displaying internal agent communication and duplicated user messages.
2. The final answer displayed to the user is cluttered with system logs and raw Spoke responses.

## Expected Behaviour

1. **Inter-Agent Delegation Schema**:
   - Define a new schema `urn:tacito:schema:conversational:agent-delegation:v1` (`SchemaConversationalAgentDelegation`).
   - The Hub MUST publish `agent-delegation` events instead of `add-user-message` when delegating tasks to Spokes.
2. **Keeper SSE Streaming**:
   - The Keeper's SSE endpoint `StreamEvents` WILL forward `agent-delegation` events.
   - Since they have a distinct schema URN, clients/UIs can easily distinguish them from user messages (`add-user-message`) and render them as coordination/delegation logs rather than user input bubbles.
3. **Payload Classification**: The `events.AgentResponsePayload` struct MUST include a `message_type` field (JSON key `message_type`).
4. **Response Classification Population**:
   - The Hub's progression/coordination updates MUST set `message_type` to `"reasoning"`.
   - Responses published by Spoke agents MUST set `message_type` to `"spoke"`.
   - The final response published by the Hub MUST set `message_type` to `"final"`.
   - Standalone single agent responses MUST set `message_type` to `"standalone"`.
5. **STM Build-up**:
   - **Spoke Agent**: When a Spoke receives `agent-delegation`, it appends it to its private STM as `"user"` and appends its reply as `"assistant"`.
   - **Hub Agent**: When the Hub receives a Spoke's response, it appends it to its private STM with the role `"user"` (formatted as `[Observation] Spoke Agent '<Name>' responded: <Response>`). This prevents the Hub's LLM from mistaking Spoke responses for its own past outputs.
6. **Clean Synthesis**: The Hub coordinator's system prompt instructions MUST guide the LLM to synthesize the Spokes' responses into a cohesive, polished final response to the user instead of copy-pasting or concatenating them directly.

## Acceptance Criteria

1. Verify that `agent-delegation` events are forwarded by the keeper's `/api/v1/events/stream` SSE stream with the correct schema segment `agent-delegation`.
2. Verify that `AgentResponsePayload` contains the new `message_type` field.
3. Verify that `message_type` is correctly set to `"reasoning"`, `"spoke"`, `"final"`, or `"standalone"` in their respective scenarios.
4. Verify that the Hub's final response synthesizes the spoke answers without raw concatenation.
5. All unit and integration tests compile and pass successfully.
