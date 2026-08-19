# Tacito Square — Architectural Foundations & Domain Model

## 1. System Overview & Philosophy

**Tacito Square** is an enterprise-grade, Kubernetes-native multi-agent platform designed to orchestrate autonomous AI agents in isolated or collaborative topologies (communities). 

Every agent in Tacito Square reasons with large language models (LLMs), executes procedural tools and Model Context Protocol (MCP) clients, maintains short-term conversational context (STM) and long-term semantic memory (LTM), and collaborates via event-driven messaging.

```
                               ┌─────────────────────────────────────────┐
                               │             BFF / Web UI                │
                               │        (GraphQL / REST / SSE)           │
                               └────────────────────┬────────────────────┘
                                                    │
                               ┌────────────────────▼────────────────────┐
                               │           Keeper Component              │
                               │  - Component & Asset Catalog            │
                               │  - Community & Agent Assembly           │
                               │  - PropagatedAgentConfig Compiler       │
                               └───────────┬─────────────────┬───────────┘
                                           │                 │
                           PostgreSQL (pgx)│                 │CRDs / NATS
                                           ▼                 ▼
                                    [( Metadata )]    ┌─────────────────────────┐
                                                      │   Kubernetes Operator   │
                                                      │  (Reconciliation Loop)  │
                                                      └───────────┬─────────────┘
                                                                  │ Spawns Pods
                     ┌────────────────────────────────────────────┼────────────────────────────────────────────┐
                     ▼                                            ▼                                            ▼
      ┌─────────────────────────────┐              ┌─────────────────────────────┐              ┌─────────────────────────────┐
      │     Agent Pod: Hub          │              │    Agent Pod: Spoke 1       │              │    Agent Pod: Spoke 2       │
      │  - LLM Brain (OpenAI API)   │   NATS       │  - LLM Brain (OpenAI API)   │   NATS       │  - LLM Brain (OpenAI API)   │
      │  - Dynamic System Prompt    │◄────────────►│  - Dynamic System Prompt    │◄────────────►│  - Dynamic System Prompt    │
      │  - MCP Tools & Skills       │  Event Bus   │  - MCP Tools & Skills       │  Event Bus   │  - MCP Tools & Skills       │
      │  - STM (Redis) / LTM(Qdrant)│              │  - STM (Redis) / LTM(Qdrant)│              │  - STM (Redis) / LTM(Qdrant)│
      └─────────────────────────────┘              └─────────────────────────────┘              └─────────────────────────────┘
```

---

## 2. Bounded Contexts & Core Components

Tacito Square is partitioned into clean, decoupled bounded contexts adhering to Hexagonal Architecture:

### A. Keeper (`cmd/keeper`, `internal/keeper`)
*   **Role**: Administrative custodianship and configuration compiler.
*   **Responsibilities**:
    *   Manages asset building blocks: Prompts, Skills, MCP Clients, LLM Bindings, and Resource Tiers.
    *   Administers Communities, Agents, and Agent Templates.
    *   Compiles declarative agent definitions into immutable `PropagatedAgentConfig` payloads.
    *   Tracks staleness markers and delivers configuration updates to the Operator.
*   **Storage**: PostgreSQL (managed strictly via `pgx/v5` and migrations with `goose/v3`).

### B. Agent Runtime (`cmd/agent`, `internal/agent`)
*   **Role**: Autonomous reasoning and execution pod.
*   **Responsibilities**:
    *   Executes LLM reasoning loops with configurable model bindings.
    *   Constructs dynamic system prompts combining role templates, community personas, and runtime context.
    *   Executes native procedural skills and Model Context Protocol (MCP) server tools.
    *   Interacts with Short-Term Memory (STM in Redis) for active thread state and locks.
    *   Queries Long-Term Memory (LTM in Qdrant) for vector embeddings and semantic recall.
    *   Communicates over NATS with subject namespacing and structured conversational envelopes.

### C. Operator (`cmd/operator`, `operator/`)
*   **Role**: Kubernetes Custom Controller.
*   **Responsibilities**:
    *   Watches `Agent` and `Community` Custom Resource Definitions (CRDs).
    *   Materializes agent pods, services, secrets, and Helm releases.
    *   Maintains reconciliation state, handles rolling updates, and enforces community scaling rules.

### D. BFF & Configurator UI (`cmd/bff`, `ui/configurator`)
*   **Role**: Client gateway and management web interface.
*   **Responsibilities**:
    *   Aggregates administrative and operational APIs.
    *   Streams Server-Sent Events (SSE) for real-time agent execution traces.
    *   Provides interactive wizard for creating, assembling, and monitoring agents and communities.

---

## 3. Topologies & Inter-Agent Coordination

### A. Standalone vs. Community Topologies
1.  **Standalone**: A single agent operating independently with direct user/API interaction.
2.  **Hub-Spoke Community**:
    *   **Hub (Coordinator)**: The primary entrypoint. Maintains high-level context, reads enriched **A2A Agent Cards**, and intelligently delegates specialized sub-tasks to Spokes.
    *   **Spokes (Specialists)**: Targeted workers with domain-specific skills (e.g. Code Reviewer, Database Analyzer, Web Searcher).
    *   **Conversation Handoff**: Hubs can delegate full conversational turns or hand off threads cleanly to specialized spokes.

### B. Agent-to-Agent (A2A) Cards
Every agent advertises a dynamic metadata card containing its identifier, role, capabilities, skills, and current availability. The Hub coordinator reads these cards to route messages without hardcoded logic.

### C. Event-Driven Messaging (NATS)
*   All inter-agent communication flows over NATS with standardized subject hierarchies:
    *   `tacito.<tenant_id>.community.<community_id>.events`
    *   `tacito.<tenant_id>.agent.<agent_id>.inbox`
    *   `tacito.<tenant_id>.agent.<agent_id>.observations`

---

## 4. Storage & Infrastructure Layer

| Subsystem | Technology | Purpose |
|---|---|---|
| **Relational Data** | PostgreSQL (`pgx/v5`) | Keeper relational store: agents, communities, templates, audit logs. |
| **Short-Term Memory** | Redis (`go-redis`) | Ephemeral session state, distributed locks, rate-limiting tokens. |
| **Long-Term Memory** | Qdrant (gRPC) | Vector embeddings, semantic similarity search, persistent knowledge. |
| **Message Broker** | NATS (`nats.go`) | High-throughput asynchronous event distribution and agent inbox queues. |
| **Tool Protocol** | MCP (`go-sdk`) | Model Context Protocol for secure external tool execution. |
| **Identity & OIDC** | Zitadel / OIDC | Multi-tenant JWT token verification and RBAC route protection. |
| **Container Runtime** | Google Distroless | `gcr.io/distroless/base-nossl-debian13` base images for minimal attack surface. |
