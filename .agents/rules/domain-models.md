---
trigger: always_on
globs: ["**/*.go", "**/*.ts", "**/*.tsx"]
description: Domain-Driven Design (DDD) principles, ubiquitous language, and domain purity standards for Tacito Square.
---

# Domain-Driven Design (DDD) & Tacito Domain Metamodel

This rule establishes the central importance of Domain-Driven Design (DDD), ubiquitous language, and domain purity across all Tacito Square components.

## 1. Domain Centrality & Pure Domain Layer

The domain layer (`internal/<component>/domain/`) is the heart of Tacito Square. It encapsulates core business logic, aggregates, value objects, domain errors, and state machines.

### Strict Purity Constraints:
- **Zero Inbound/Outbound Adapter Imports**: The domain layer MUST NOT import packages from `adapters/` (e.g. pgx, gin, redis, nats, openai, qdrant) or `application/`.
- **Zero Database / Driver Types in Domain**: Never use `sql.NullString`, `pgtype.UUID`, `json.RawMessage`, or ORM struct tags in domain entities. Use native Go types (`string`, `uuid.UUID`, custom Value Objects).
- **Zero Framework Annotations**: Domain models must not contain HTTP binding tags (`binding:"required"`) or validation tags tied to transport libraries. Domain validation belongs in domain methods (`func (a Agent) Validate() error`).
- **Sentinel Domain Errors**: Export domain-meaningful errors (`var ErrAgentNotFound = errors.New("agent not found")`) and custom domain error types rather than raw database or RPC errors.

## 2. Ubiquitous Language & Bounded Contexts

Always use the precise domain terms defined in the Tacito Square ubiquitous language:

### A. Keeper Administrative Context
- **Asset / Building Block**: A modular configuration component administered by Keeper.
  - `PromptTemplate`: Parametric instruction snippet, persona, or guideline.
  - `Skill`: A procedural capability or tool invocation schema.
  - `MCPClient`: Connection definition for an external Model Context Protocol server.
  - `LLMBinding`: Configuration binding to an external model provider (OpenAI, Ollama, Anthropic).
  - `ResourceTier`: Compute and memory ceiling profile.
- **Collections & Bundles**: Groupings of assets (e.g. `PromptCollection`, `SkillCollection` / `Skillset`).
- **Templates vs Instances**:
  - `AgentTemplate`: Reusable, parameterized blueprint for an agent archetype.
  - `Agent`: Configured agent entity within a tenant.
  - `Community`: Collaborative boundary aggregating multiple assigned agents with a defined topology (`standalone`, `hub_spoke`).
  - `AgentAssignment`: The association binding an agent to a community with a specific role (`hub`, `spoke`, `standalone`).
  - `PropagatedAgentConfig`: The compiled, immutable, fully-resolved snapshot delivered to the Kubernetes Operator.

### B. Agent Runtime Context
- `CognitiveEngine`: Core LLM reasoning loop executing turns, managing tool execution, and emitting observations.
- `SystemPromptPipeline`: Dynamic assembly pipeline that compiles system prompts from role templates, community personas, and runtime context.
- `A2A Agent Card`: Dynamic advertisement of an agent's capabilities, skills, and availability used by Hub coordinators for routing.
- `Short-Term Memory (STM)`: Redis-backed active thread context, distributed locks, and conversational scratchpad.
- `Long-Term Memory (LTM)`: Qdrant-backed vector embeddings and semantic recall for cross-session knowledge.
- `Event Envelope`: Standardized message envelope carrying tenant, trace context, timestamp, and domain payload over NATS.

### C. Kubernetes Operator Context
- `Agent` & `Community` CRDs: Kubernetes Custom Resources declaring the desired state of agents and communities.
- `Reconciliation Loop`: Controller loop that materializes Kubernetes deployments, pods, ConfigMaps, and secrets to match `PropagatedAgentConfig`.

---

## Developer Checklists & Verifications

- [ ] Does my domain model package (`domain/model/`) contain ANY imports of database, redis, gin, or nats packages? (Must be ZERO).
- [ ] Are domain entities using pure Go types instead of driver-specific types?
- [ ] Are error conditions returned as domain sentinel errors?
- [ ] Do variable names and method signatures use the ubiquitous language (e.g. `PropagatedAgentConfig`, `AgentAssignment`, `CognitiveEngine`)?
