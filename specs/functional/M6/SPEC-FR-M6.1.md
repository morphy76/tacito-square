# SPEC-FR-M6.1: Community Topology (Single Agent & Hub-Spoke)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M6.1                                |
| Status        | DRAFT                                       |
| Milestone     | M6                                          |
| Component     | keeper, agent, operator                     |
| Depends On    | SPEC-FR-M3.6, SPEC-FR-M4.2                  |
| Supersedes    | none                                        |

## Context

Communities organize agents in topologies. The system supports two topologies:
1. **Single Agent (Default)**: A simplified layout containing exactly one standalone agent.
2. **Hub-Spoke**: An advanced orchestration layout with one Hub agent coordinating multiple specialized Spoke agents.

---

## Specification

### 1. Topology & Role Configuration

#### A. Keeper Database Schema
* The Keeper `communities` table MUST include a `topology` column:
  * Type: `VARCHAR` (e.g., `single-agent` or `hub-spoke`).
  * Default: `single-agent`.
* The Keeper `agents` table MUST include a `role` column (VARCHAR, default `'spoke'`).
* **Validation & Mutability Rules**:
  * **Topology Mutability**:
    * A community's topology type (`single-agent` or `hub-spoke`) CAN ONLY be updated/switched if there are currently zero agents assigned to that community.
    * Once one or more agents are assigned to a community, its topology type becomes immutable.
  * **Single Agent Topology**:
    * The community MUST contain exactly one agent.
    * No other agents can be assigned to the community.
    * The single agent behaves as a standard standalone agent (the `role` field can be omitted or ignored).
  * **Hub-Spoke Topology**:
    * The community MUST contain exactly one agent with `role = 'hub'`.
    * The community can contain multiple agents with `role = 'spoke'`.
    * A community cannot be transitioned to a deployed/active status if it does not satisfy this constraint.

#### B. Kubernetes API (TacitoAgent CRD)
* The `TacitoAgentSpec` MUST include a new `role` field:
  ```go
  // Role defines the topological role of the agent in the community.
  // Must be either "hub" or "spoke". Defaults to "spoke".
  // +kubebuilder:validation:Enum=hub;spoke
  // +kubebuilder:default=spoke
  // +optional
  Role string `json:"role,omitempty"`
  ```
* The Operator MUST read `spec.role` and propagate it to the agent container as the `TS_AGENT_ROLE` environment variable.

---

### 2. Messaging Entrypoints & Routing

#### A. Single Agent Topology (Default)
* **Propagation**: At this stage, agents are not directly exposed. The system uses the existing free event API in the Keeper to propagate conversation events to the single agent's NATS subject: `ts.community.{community_id}.agent.{agent_id}`.
* **No Orchestration**: The agent processes the message directly and publishes the final `SchemaConversationalAgentResponse` on its response subject.

#### B. Hub-Spoke Topology
* **Hub Entrypoint**: The Hub agent is the designated entrypoint for all inbound community messages.
* **Inbound Traffic**: The BFF/Keeper publishes conversation events to the Hub's NATS subject: `ts.community.{community_id}.agent.hub`.
* **Asynchronous Orchestration**: 
  * The Hub receives the request, stores the orchestration state to Redis, and delegates tasks to Spoke agents by publishing to: `ts.community.{community_id}.agent.{spoke_id}`.
  * Spokes process tasks and publish completion events back to the Hub's response subject.
  * The Hub handles the responses (which can be load-balanced across Hub replicas via a NATS Queue Group) using an Asynchronous State Machine, continuing coordination or returning the final response to the BFF/Keeper.

---

### 3. Hub Agent Specifics & Replica Coordination

* **Configurable Brain**: The Hub agent runs the standard `tacito-agent` container image with `TS_AGENT_ROLE=hub`. Its brain remains fully configurable.
* **Router Prompt**: Creating a hub can leverage the preconfigured system prompt from the prompt catalog, but the user configurator can optionally specify a custom router prompt.
* **Agent Cards Discovery**: The Hub agent discovers and maps tasks to Spokes dynamically by reading their published capabilities profile ("Agent Cards").
* **Replica Management**: Hub replicas use a NATS Queue Group (e.g., `hub-queue-group`) for the inbound subject to prevent duplicate processing. All active orchestration states are stored statelessly in Redis, coordinated via Redis distributed locks to prevent race conditions on parallel execution turns.

---

## Acceptance Criteria

1. Keeper's database migrations add the `topology` column to the `communities` table and the `role` column to the `agents` table.
2. Keeper REST APIs enforce the validation rules for both `single-agent` and `hub-spoke` topologies.
3. In `single-agent` mode, the BFF publishes messages directly to the standalone agent's NATS subject.
4. In `hub-spoke` mode, the BFF publishes messages to the Hub agent's NATS subject.
5. The Hub agent starts up in `hub` mode when `TS_AGENT_ROLE=hub` is set, executing the asynchronous state machine.

## Test Plan

* **Unit Tests**:
  * Verify validation constraints in the Keeper API for both topologies.
  * Verify CRD validation for the agent `spec.role` field.
* **Integration Tests**:
  * Verify direct end-to-end messaging for a community using `single-agent` topology.
  * Verify load-balanced, asynchronous inter-agent routing for a community using `hub-spoke` topology with multiple Hub replicas.

## Files Affected

* `pkg/kubernetes/apis/tacito/v1alpha1/tacitoagent_types.go`
* `internal/operator/application/service/reconcile_service.go`
* `internal/keeper/domain/model/community.go`
* `internal/keeper/domain/model/agent.go`
* `internal/agent/adapters/inbound/nats/event_subscriber.go`
* `internal/agent/application/service/cognitive_engine.go`


