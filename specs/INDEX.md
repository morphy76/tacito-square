# Tacito Square — Spec Index

> Single source of truth for all specs. Generated from `specs/` directory.

## Non-Functional Requirements & Agent Rules

Non-functional requirements (NFRs) are codified as **Agent Rules** inside `.agents/rules/` and are actively enforced during development:

| Rule | Title & Description | Target Glob | Superseded Specs |
|------|---------------------|-------------|------------------|
| [spec_driven_development](file:///Users/R.Pasquini/Projects/side/tacito-square/.agents/rules/spec-driven-development.md) | **Spec-Driven & Test-Driven (TDD) Guidelines**: Enforces strict "no code without functional spec" rule, task tracking, and Red/Green/Refactor loops. | `**/*.{go,ts,md}` | — |
| [cloud_first](file:///Users/R.Pasquini/Projects/side/tacito-square/.agents/rules/cloud-first.md) | **Cloud-First & Multitenancy Guidelines**: Ephemeral design, circuit breakers, retries, statelessness, multitenancy resolution, API-first and contract-based isolation. | `**/*.{go,ts}` | `SPEC-NFR-CLOUD`, `SPEC-NFR-MULTITENANCY`, `SPEC-NFR-OPENAPI` |
| [k8s_best_practices](file:///Users/R.Pasquini/Projects/side/tacito-square/.agents/rules/k8s-best-practices.md) | **Kubernetes Best Practices**: Horizontal Pod Autoscaling (HPA) templates, dependency-aware health probes, and distroless container images. | `**/*.{go,ts,yaml,Dockerfile}` | `SPEC-NFR-HPA`, `SPEC-NFR-HEALTH`, `SPEC-NFR-STACK` (Docker base) |
| [observability](file:///Users/R.Pasquini/Projects/side/tacito-square/.agents/rules/observability.md) | **Observability Standards**: Prometheus metric endpoints, OpenTelemetry traces, and zerolog/winston structured JSON logging. | `**/*.{go,ts}` | `SPEC-NFR-OBSERVABILITY`, `SPEC-NFR-LOG` |
| [code_architecture](file:///Users/R.Pasquini/Projects/side/tacito-square/.agents/rules/code-architecture.md) | **Code Architecture Guidelines**: Hexagonal (Ports & Adapters) domain boundaries and reactive Go concurrency primitives. | `**/*.{go,ts}` | `SPEC-NFR-HEXAGONAL`, `SPEC-NFR-REACTIVE` |
| [nonfunctional](file:///Users/R.Pasquini/Projects/side/tacito-square/.agents/rules/nonfunctional.md) | **General NFR & Stack Constraints**: Approved library locks, monorepo Makefiles, SemVer component release lifecycles, and Gin conventions. | `*` | `SPEC-NFR-STACK`, `SPEC-NFR-BUILDING`, `SPEC-NFR-VERSIONING`, `SPEC-NFR-HTTP` |


## Functional Requirement Specs

### M1, M2 & M3 Summary

Milestones 1, 2, and 3 have been completed and consolidated into a single summary. For the complete list of specifications and resolved bugs, refer to the [Milestones M1, M2 & M3 Summary](milestones/M1-M2-M3-summary.md).


### M4 Summary

Milestone 4 has been completed and consolidated. For the complete list of specifications and resolved bugs, refer to the [Milestone M4: Operator Core Summary](milestones/M4-summary.md).


### M5 Summary

Milestone 5 has been completed and consolidated. For the complete list of specifications and resolved bugs, refer to the [Milestone M5: Agent Core Summary](milestones/M5-summary.md).




### M6: Communities & Messaging

| ID | Title | Status | Component | File |
|----|-------|--------|-----------|------|
| SPEC-FR-M6.0 | Event-Driven Architecture Foundation & Conversational Schema | VERIFIED | shared, keeper, agent | [SPEC-FR-M6.0](functional/M6/SPEC-FR-M6.0.md) |
| SPEC-FR-M6.1 | Community Topology (Hub-Spoke) | VERIFIED | keeper, agent | [SPEC-FR-M6.1](functional/M6/SPEC-FR-M6.1.md) |
| SPEC-FR-M6.2 | NATS Inter-Agent Messaging | VERIFIED | agent | [SPEC-FR-M6.2](functional/M6/SPEC-FR-M6.2.md) |
| SPEC-FR-M6.3 | NATS Subject Namespacing | VERIFIED | agent, keeper | [SPEC-FR-M6.3](functional/M6/SPEC-FR-M6.3.md) |
| SPEC-FR-M6.5 | A2A Agent Cards | VERIFIED | agent, keeper | [SPEC-FR-M6.5](functional/M6/SPEC-FR-M6.5.md) |
| SPEC-FR-M6.6 | Conversation Handoff | VERIFIED | agent | [SPEC-FR-M6.6](functional/M6/SPEC-FR-M6.6.md) |

### M6: Bugs

| ID | Title | Status | Severity | File |
|----|-------|--------|----------|------|
| BUG-M6.1 | Agent Cards dynamically empty or missing actual values on startup | CLOSED | MEDIUM | [BUG-M6.1](tasks/M6.BUG1/BUG-M6.1.md) |
| BUG-M6.2 | Unassigning agent from community does not evict registration or cards cache | CLOSED | HIGH | [BUG-M6.2](tasks/M6.BUG2/BUG-M6.2.md) |
| BUG-M6.3 | Inbound community events are routed to all spokes in hub-spoke topology instead of the Hub coordinator | CLOSED | HIGH | [BUG-M6.3](tasks/M6.BUG3/BUG-M6.3.md) |
| BUG-M6.4 | Redundant Assignment or Unassignment Fails to Reconcile Deployment Status | CLOSED | MEDIUM | [BUG-M6.4](tasks/M6.BUG4/BUG-M6.4.md) |
| BUG-M6.5 | Hub Agent Deployed with Role 'spoke' in Hub-Spoke Community | CLOSED | HIGH | [BUG-M6.5](tasks/M6.BUG5/BUG-M6.5.md) |
| BUG-M6.6 | Orchestrator Loop Limit Terminating Threads Instead of Returning Latest Spoke Response | CLOSED | HIGH | [BUG-M6.6](tasks/M6.BUG6/BUG-M6.6.md) |
| BUG-M6.7 | Agent Messages Pollution and Lack of Classification in Server-Sent Events (SSE) Stream | CLOSED | HIGH | [BUG-M6.7](tasks/M6.BUG7/BUG-M6.7.md) |
| BUG-M6.8 | Hardcoded Hub System Prompt and Lack of Template Parameterization | CLOSED | HIGH | [BUG-M6.8](tasks/M6.BUG8/BUG-M6.8.md) |
| BUG-M6.9 | Agent Brain Embeds LLM Binding Instead of Referring LLM Binding Model | CLOSED | HIGH | [BUG-M6.9](tasks/M6.BUG9/BUG-M6.9.md) |
| BUG-M6.10 | Hub-Spoke Orchestration Observation Role Confusion and Loop | CLOSED | HIGH | [BUG-M6.10](tasks/M6.BUG10/BUG-M6.10.md) |
| BUG-M6.11 | Spoke Response CoT Leakage due to Missing Polishing on Delegated turns | CLOSED | MEDIUM | [BUG-M6.11](tasks/M6.BUG11/BUG-M6.11.md) |
| BUG-M6.12 | Unsafe Distributed Lock Implementation | CLOSED | HIGH | [BUG-M6.12](tasks/M6.BUG12/BUG-M6.12.md) |
| BUG-M6.13 | Missing Self-Delegation Runtime Guard | CLOSED | MEDIUM | [BUG-M6.13](tasks/M6.BUG13/BUG-M6.13.md) |
| BUG-M6.14 | Handoff Target Not Found Missing Observability | CLOSED | LOW | [BUG-M6.14](tasks/M6.BUG14/BUG-M6.14.md) |
| BUG-M6.15 | OrchestrationState Status Uses Raw Strings | CLOSED | LOW | [BUG-M6.15](tasks/M6.BUG15/BUG-M6.15.md) |



### M7: BFF & UIs

| ID | Title | Status | Component | File |
|----|-------|--------|-----------|------|
| SPEC-FR-M7.1 | BFF API Bridge Layer | VERIFIED | bff | [SPEC-FR-M7.1](functional/M7/SPEC-FR-M7.1.md) |
| SPEC-FR-M7.2 | Configurator UI | ACCEPTED | ui | [SPEC-FR-M7.2](functional/M7/SPEC-FR-M7.2.md) |
| SPEC-FR-M7.3 | Auditor UI | ACCEPTED | ui | [SPEC-FR-M7.3](functional/M7/SPEC-FR-M7.3.md) |
| SPEC-FR-M7.4 | OIDC Login Flow (Keycloak) | ACCEPTED | bff | [SPEC-FR-M7.4](functional/M7/SPEC-FR-M7.4.md) |
| SPEC-FR-M7.5 | Standardize query parameters for GET APIs | DRAFT | keeper, bff | [SPEC-FR-M7.5](functional/M7/SPEC-FR-M7.5.md) |
| SPEC-FR-M7.6 | BFF primary API surface to use GraphQL | DRAFT | bff, ui | [SPEC-FR-M7.6](functional/M7/SPEC-FR-M7.6.md) |
| SPEC-FR-M7.7 | Serve Welcome index.html in BFF | VERIFIED | bff | [SPEC-FR-M7.7](functional/M7/SPEC-FR-M7.7.md) |

### M8: RAG & LTM

| ID | Title | Status | Component | File |
|----|-------|--------|-----------|------|
| SPEC-FR-M8.1 | Design architectural integration patterns for RAG and web search | DRAFT | agent, shared | [SPEC-FR-M8.1](functional/M8/SPEC-FR-M8.1.md) |

### M9: Governance

| ID | Title | Status | Component | File |
|----|-------|--------|-----------|------|
| SPEC-FR-M9.1 | OIDC/JWT Authentication | DRAFT | keeper, shared | [SPEC-FR-M9.1](functional/M9/SPEC-FR-M9.1.md) |
| SPEC-FR-M9.2 | RBAC Role Model & Route Protection | DRAFT | keeper, shared | [SPEC-FR-M9.2](functional/M9/SPEC-FR-M9.2.md) |
| SPEC-FR-M9.3 | Usage Quotas (community + agent) | DRAFT | keeper | [SPEC-FR-M9.3](functional/M9/SPEC-FR-M9.3.md) |
| SPEC-FR-M9.4 | Quota Enforcement (Redis counters) | DRAFT | keeper | [SPEC-FR-M9.4](functional/M9/SPEC-FR-M9.4.md) |
| SPEC-FR-M9.5 | HITL Yield & Callback Flows | DRAFT | agent, keeper | [SPEC-FR-M9.5](functional/M9/SPEC-FR-M9.5.md) |
| SPEC-FR-M9.6 | Audit Trail (events + queries) | DRAFT | keeper | [SPEC-FR-M9.6](functional/M9/SPEC-FR-M9.6.md) |
| SPEC-FR-M9.7 | Prompt Management (CRUD + versioning) | DRAFT | keeper | [SPEC-FR-M9.7](functional/M9/SPEC-FR-M9.7.md) |
| SPEC-FR-M9.8 | Skills Management (CRUD + MCP attach) | DRAFT | keeper | [SPEC-FR-M9.8](functional/M9/SPEC-FR-M9.8.md) |
| SPEC-FR-M9.9 | Zero-Scaling Support | ACCEPTED | operator | [SPEC-FR-M9.9](functional/M9/SPEC-FR-M9.9.md) |
| SPEC-FR-M9.10 | Continuous Integration (GitHub Actions) | ACCEPTED | build | [SPEC-FR-M9.10](functional/M9/SPEC-FR-M9.10.md) |
| SPEC-FR-M9.11 | Encrypt agent brain credential secrets at rest | DRAFT | keeper | [SPEC-FR-M9.11](functional/M9/SPEC-FR-M9.11.md) |
| SPEC-FR-M9.12 | Integrate Unleash feature flag management | DRAFT | shared, keeper, bff, agent | [SPEC-FR-M9.12](functional/M9/SPEC-FR-M9.12.md) |
| SPEC-FR-M9.13 | Track brain token usage per agent and thread | DRAFT | agent, keeper | [SPEC-FR-M9.13](functional/M9/SPEC-FR-M9.13.md) |
| SPEC-FR-M9.14 | Provide APIs to manage tenant secrets for LLM bindings | DRAFT | keeper, bff | [SPEC-FR-M9.14](functional/M9/SPEC-FR-M9.14.md) |
| SPEC-FR-M9.15 | Create a Skillset abstraction to group multiple skills | DRAFT | keeper, shared, agent | [SPEC-FR-M9.15](functional/M9/SPEC-FR-M9.15.md) |
| SPEC-FR-M9.16 | Provide configuration flags to opt-out of specific built-in tools | DRAFT | keeper, agent | [SPEC-FR-M9.16](functional/M9/SPEC-FR-M9.16.md) |
| SPEC-FR-M9.17 | Introduce template engines for brain prompts | DRAFT | agent, shared | [SPEC-FR-M9.17](functional/M9/SPEC-FR-M9.17.md) |

### M9: Bugs

| ID | Title | Status | Severity | File |
|----|-------|--------|----------|------|
| BUG-M9.1 | Missing Horizontal Pod Autoscaler in Standalone Agent Chart | OPEN | MEDIUM | [BUG-M9.1](tasks/M9.BUG1/BUG-M9.1.md) |

### M10: Hardening

| ID | Title | Status | Component | File |
|----|-------|--------|-----------|------|
| SPEC-FR-M10.1 | Prometheus Metrics Integration | DRAFT | all | [SPEC-FR-M10.1](functional/M10/SPEC-FR-M10.1.md) |
| SPEC-FR-M10.2 | OpenAPI Contract Validation | DRAFT | all | [SPEC-FR-M10.2](functional/M10/SPEC-FR-M10.2.md) |
| SPEC-FR-M10.3 | E2E & Benchmark Tests | DRAFT | test | [SPEC-FR-M10.3](functional/M10/SPEC-FR-M10.3.md) |
| SPEC-FR-M10.4 | Production Helm & Hardening | DRAFT | deploy | [SPEC-FR-M10.4](functional/M10/SPEC-FR-M10.4.md) |
| SPEC-FR-M10.5 | K8s NetworkPolicies | DRAFT | operator | [SPEC-FR-M10.5](functional/M10/SPEC-FR-M10.5.md) |
| SPEC-FR-M10.6 | Comprehensive System Documentation | DRAFT | docs | [SPEC-FR-M10.6](functional/M10/SPEC-FR-M10.6.md) |
| SPEC-FR-M10.7 | Benchmark Suite & Integration Coverage Verification | ACCEPTED | test | [SPEC-FR-M10.7](functional/M10/SPEC-FR-M10.7.md) |

### M10: Bugs

| ID | Title | Status | Severity | File |
|----|-------|--------|----------|------|
| BUG-M10.1 | Infrastructure Services Do Not Enforce SSL/TLS or Authenticated Connections | OPEN | HIGH | [BUG-M10.1](tasks/M10.BUG1/BUG-M10.1.md) |

### M11: Federation

| ID | Title | Status | Component | File |
|----|-------|--------|-----------|------|
| SPEC-FR-M11.1 | A2A HTTP Gateway | DRAFT | keeper | [SPEC-FR-M11.1](functional/M11/SPEC-FR-M11.1.md) |
| SPEC-FR-M11.2 | External Agent Registry | DRAFT | keeper | [SPEC-FR-M11.2](functional/M11/SPEC-FR-M11.2.md) |
| SPEC-FR-M11.3 | Spawning MCP Servers using CRD from Keeper | DRAFT | keeper, operator | [SPEC-FR-M11.3](functional/M11/SPEC-FR-M11.3.md) |
| SPEC-FR-M11.4 | Thread Management | DRAFT | conversation-hub | [SPEC-FR-M11.4](functional/M11/SPEC-FR-M11.4.md) |

### M12: Extensions & Advanced Topologies

| ID | Title | Status | Component | File |
|----|-------|--------|-----------|------|
| SPEC-FR-M12.1 | Decentralized P2P Topology & Handoff | DRAFT | agent, keeper, operator | [SPEC-FR-M12.1](functional/M12/SPEC-FR-M12.1.md) |
| SPEC-FR-M12.2 | Shared Hive Short-Term Memory (Community STM) | DRAFT | agent | [SPEC-FR-M12.2](functional/M12/SPEC-FR-M12.2.md) |
| SPEC-FR-M12.3 | Specialist Agent Spawn | DRAFT | keeper, agent | [SPEC-FR-M12.3](functional/M12/SPEC-FR-M12.3.md) |
| SPEC-FR-M12.4 | Built-in Agent Templates | ACCEPTED | keeper, agent, operator | [SPEC-FR-M12.4](functional/M12/SPEC-FR-M12.4.md) |

## Milestones

| Milestone | Title | Status | Specs | File |
|-----------|-------|--------|-------|------|
| M1 | Infrastructure Helm Chart | ✔️ IMPLEMENTED | 2 | [M1-M2-M3 Summary](milestones/M1-M2-M3-summary.md) |
| M2 | Application Helm Chart & Component Scaffolding | ✔️ IMPLEMENTED | 10 | [M1-M2-M3 Summary](milestones/M1-M2-M3-summary.md) |
| M3 | Keeper Core | ✔️ IMPLEMENTED | 8 | [M1-M2-M3 Summary](milestones/M1-M2-M3-summary.md) |
| M4 | Operator Core | ✔️ IMPLEMENTED | 5 | [M4 Summary](milestones/M4-summary.md) |
| M5 | Agent Core | ✔️ IMPLEMENTED | 9 | [M5 Summary](milestones/M5-summary.md) |
| M6 | Communities & Messaging | ⏳ IN_PROGRESS | 6 | [M6](milestones/M6-communities.md) |
| M7 | BFF & UIs | ⬜ PLANNED | 6 | [M7](milestones/M7-bff-uis.md) |
| M8 | RAG & LTM | ⬜ PLANNED | 1 | [M8](milestones/M8-rag-ltm.md) |
| M9 | Governance | ⬜ PLANNED | 17 | [M9](milestones/M9-governance.md) |
| M10 | Hardening | ⬜ PLANNED | 8 | [M10](milestones/M10-hardening.md) |
| M11 | Federation | ⬜ PLANNED | 4 | [M11](milestones/M11-federation.md) |
| M12 | Extensions & Advanced Topologies | ⬜ PLANNED | 4 | [M12](milestones/M12-extensions.md) |

