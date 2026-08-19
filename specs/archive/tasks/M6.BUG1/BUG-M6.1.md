# BUG-M6.1: Agent Cards dynamically empty or missing actual values on startup

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M6.1                                                           |
| Status        | CLOSED                                                             |
| Severity      | MEDIUM                                                             |
| Milestone     | M6 — Communities & Messaging                                       |
| Affects       | `internal/agent/adapters/outbound/nats/heartbeat.go`               |
| Violates      | SPEC-FR-M6.5                                                       |
| Discovered    | Manual audit of agent heartbeat message content on startup         |

## Problem Statement

When the Agent starts up and compiles its `AgentCard` for the heartbeat publisher, it maps configuration keys directly from Viper. However, in Kubernetes deployment mode:
1. Environment variables like `TS_AGENT_DESCRIPTION` or `TS_AGENT_URL` are not injected by the Operator.
2. The dynamic database-configured description and dynamic skills are instead propagated inside `TS_AGENT_SYSTEM_PROMPT` as a structured JSON string (`PropagatedAgentConfig`).
3. The `compileCard` method does not parse this system prompt JSON to extract the agent's description and dynamic skills.
4. As a result, the compiled heartbeat card is missing the agent's actual business description and dynamic skills list, sending empty/default values to the Keeper registry.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| Agent Heartbeat Publisher | `internal/agent/adapters/outbound/nats/heartbeat.go` | `compileCard` does not parse the `TS_AGENT_SYSTEM_PROMPT` JSON fallback to populate description and dynamic skills. |

## Impact

1. The central agent registry in the Keeper receives incomplete Agent Cards lacking descriptions and dynamic skills.
2. Capability-based routing and agent discovery within the community will fail since downstream agents cannot see the actual skills and description of running agents.

## Expected Behaviour

1. If `TS_AGENT_SYSTEM_PROMPT` is a valid JSON matching `PropagatedAgentConfig`:
   - The agent card `Description` MUST be populated from the JSON's `description` field (if not explicitly set in other configuration overrides).
   - The agent card `Skills` MUST append all dynamic skills parsed from the JSON's `skills` list.
2. The agent card `Capabilities` must also map `pushNotifications` and `stateTransitionHistory` if set.
3. The agent card `URL` must have a sensible fallback (e.g. `http://localhost:<port>`) if not configured, to satisfy A2A schema requirement.

## Acceptance Criteria

1. A unit test replicates the missing description and skills issue by injecting a structured system prompt, verifying that the compiled card correctly extracts description and dynamic skills (RED Phase).
2. The `compileCard` function is updated to parse structured system prompts and populate capabilities and URL fallbacks (GREEN Phase).
3. The tests pass successfully.
