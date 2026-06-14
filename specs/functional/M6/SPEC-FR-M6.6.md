# SPEC-FR-M6.6: Conversation Handoff

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M6.6                                |
| Status        | DRAFT                                       |
| Milestone     | M6                                          |
| Component     | agent                                       |
| Depends On    | SPEC-FR-M6.1, SPEC-FR-M5.3, SPEC-FR-M6.0    |
| Supersedes    | none                                        |

## Context

During a conversational reasoning loop, an agent may determine that another agent within the community is better suited to answer the user's request. Handoff facilitates transferring the conversation thread, reason, and short-term memory (STM) history to the target agent.

This specification details handoff mechanics under both coordinated (Hub-Spoke) and decentralized communities.

## Specification

### 1. Topology Specific Mechanics

#### A. Coordinated (Hub-Spoke) Topology
- Spoke agents MUST NOT initiate peer-to-peer handoffs directly.
- If a Spoke agent determines that another agent is better suited, it MUST return a structured response back to the Hub (via the response subject) detailing:
  - The suggested target agent.
  - The context, task description, or reason for handoff.
- The Hub agent's orchestrator acts as the sole controller:
  - It handles the Spoke's suggestion within its reasoning loop.
  - If approved, the Hub updates its orchestration state in Redis and publishes the new delegation task to the target Spoke.

#### B. Decentralized Peer-to-Peer Topology
- In communities without a central Hub coordinator, the source agent initiates a direct handoff.
- The source agent publishes a `handoff` event payload (`urn:tacito:schema:conversational:handoff:v1`) to NATS subject:
  `ts.community.{community_id}.agent.{target_agent_id}`
- The target agent MUST acquire the thread lock using `ThreadLock` to prevent race conditions before processing the handoff.
- The target agent processes the handoff event, appends a system metadata notice to STM, and initiates its reasoning pipeline.

### 2. Short-Term Memory (STM) Alignment
- To avoid physical data migration of Redis keys during handoff, the Redis keyspace for Short-Term Memory MUST be scoped at the community level rather than the agent level:
  - **Old Key Format**: `ts:stm:{tenant_id}:{agent_id}:{thread_id}`
  - **New Key Format**: `ts:stm:{tenant_id}:{community_id}:{thread_id}`
- All agents in a community can read and write to the same thread history, utilizing the `Role` or `AgentName` fields within memory entries to track which agent uttered which response.

### 3. Handoff Event Schema Ref
- Define constant `SchemaConversationalHandoff = "urn:tacito:schema:conversational:handoff:v1"`.
- The event payload MUST include:
  ```json
  {
    "thread_id": "uuid",
    "community_id": "uuid",
    "source_agent": "agent-a",
    "target_agent": "agent-b",
    "reason": "Reason details for handoff"
  }
  ```

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.

