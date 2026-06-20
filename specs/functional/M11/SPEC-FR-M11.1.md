# SPEC-FR-M11.1: Decentralized P2P Topology & Handoff

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M11.1                               |
| Status        | DRAFT                                       |
| Milestone     | M11                                         |
| Component     | agent, keeper, operator                     |
| Depends On    | SPEC-FR-M6.1, SPEC-FR-M6.6                  |
| Supersedes    | none                                        |

## Context

To extend the baseline topology options, communities must support a decentralized model where agents coordinate directly with each other without a central Hub. Direct peer-to-peer (P2P) handoffs allow the active conversation thread ownership to shift dynamically between agents.

## Specification

### 1. Decentralized Topology Support

1. **Keeper Database & API**:
   * Allow `decentralized` as a valid enum value in the community `topology` column.
   * In a decentralized topology, any number of agents can be assigned to the community without role constraints (roles like `hub` or `spoke` are ignored or treated as equal peers).
2. **Kubernetes API & Operator**:
   * Update the Operator to support deploying communities in decentralized mode, mounting matching configuration variables (`TS_AGENT_TOPOLOGY=decentralized`).

### 2. Direct Peer-to-Peer Handoff Flow

In a decentralized community, agents communicate directly to route threads:

1. **Direct Handoff Event**:
   * When an agent decides to hand off a conversation, it publishes a `handoff` event (`urn:tacito:schema:conversational:handoff:v1`) directly to NATS.
   * NATS subject format: `ts.community.{community_id}.agent.{target_agent_id}`
2. **Thread Locking**:
   * Since there is no central hub coordination, peer agents must acquire a thread lock before processing the handoff to prevent race conditions or simultaneous execution on the same thread.
3. **Execution Routing**:
   * The target agent processes the handoff event, appends a system metadata notification to its short-term memory, and starts its reasoning loop to respond directly back to the user or BFF gateway.

## Acceptance Criteria

To be defined during Milestone 11 review.

## Test Plan

To be defined during Milestone 11 review.

## Files Affected

To be defined during Milestone 11 review.
