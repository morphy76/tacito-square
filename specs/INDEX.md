# Tacito Square — Spec Index

> Single source of truth for all specs. Generated from `specs/` directory.

## Non-Functional Requirement Specs

| ID | Title | Status | File |
|----|-------|--------|------|
| SPEC-NFR-HEXAGONAL | Hexagonal Architecture with DDD | ACCEPTED | [SPEC-NFR-HEXAGONAL](nonfunctional/SPEC-NFR-HEXAGONAL.md) |
| SPEC-NFR-CLOUD | Cloud-First Patterns | ACCEPTED | [SPEC-NFR-CLOUD](nonfunctional/SPEC-NFR-CLOUD.md) |
| SPEC-NFR-REACTIVE | Reactive Programming | ACCEPTED | [SPEC-NFR-REACTIVE](nonfunctional/SPEC-NFR-REACTIVE.md) |
| SPEC-NFR-STACK | Technology Stack | ACCEPTED | [SPEC-NFR-STACK](nonfunctional/SPEC-NFR-STACK.md) |
| SPEC-NFR-LOG | Structured Logging (zerolog) | ACCEPTED | [SPEC-NFR-LOG](nonfunctional/SPEC-NFR-LOG.md) |
| SPEC-NFR-HTTP | HTTP Framework (Gin) | ACCEPTED | [SPEC-NFR-HTTP](nonfunctional/SPEC-NFR-HTTP.md) |
| SPEC-NFR-HEALTH | Dependency-Aware Health Probes | ACCEPTED | [SPEC-NFR-HEALTH](nonfunctional/SPEC-NFR-HEALTH.md) |
| SPEC-NFR-OBSERVABILITY | Observability (Metrics, Tracing, and Correlation) | ACCEPTED | [SPEC-NFR-OBSERVABILITY](nonfunctional/SPEC-NFR-OBSERVABILITY.md) |
| SPEC-NFR-OPENAPI | Live OpenAPI Spec Endpoints | ACCEPTED | [SPEC-NFR-OPENAPI](nonfunctional/SPEC-NFR-OPENAPI.md) |
| SPEC-NFR-HPA | Horizontal Pod Autoscaling | ACCEPTED | [SPEC-NFR-HPA](nonfunctional/SPEC-NFR-HPA.md) |
| SPEC-NFR-CACHE | Redis Infrastructure Cache | ACCEPTED | [SPEC-NFR-CACHE](nonfunctional/SPEC-NFR-CACHE.md) |
| SPEC-NFR-BUILDING | Build System | ACCEPTED | [SPEC-NFR-BUILDING](nonfunctional/SPEC-NFR-BUILDING.md) |
| SPEC-NFR-VERSIONING | Component Versioning & Lifecycle | ACCEPTED | [SPEC-NFR-VERSIONING](nonfunctional/SPEC-NFR-VERSIONING.md) |
| SPEC-NFR-MULTITENANCY | Multitenancy Architecture | ACCEPTED | [SPEC-NFR-MULTITENANCY](nonfunctional/SPEC-NFR-MULTITENANCY.md) |

## Functional Requirement Specs

### M1: Infrastructure Helm Chart

| ID | Title | Status | Component | File |
|----|-------|--------|-----------|------|
| SPEC-FR-M1.1 | Infrastructure Helm Chart | IMPLEMENTED | deploy | [SPEC-FR-M1.1](functional/M1/SPEC-FR-M1.1.md) |
| SPEC-FR-M1.2 | Makefile Infrastructure Targets | IMPLEMENTED | build | [SPEC-FR-M1.2](functional/M1/SPEC-FR-M1.2.md) |

### M2: Application Helm Chart & Component Scaffolding

| ID | Title | Status | Component | File |
|----|-------|--------|-----------|------|
| SPEC-FR-M2.1 | Application Helm Chart (infra-free, binding interfaces) | IMPLEMENTED | deploy | [SPEC-FR-M2.1](functional/M2/SPEC-FR-M2.1.md) |
| SPEC-FR-M2.2 | Shared Foundation Library | IMPLEMENTED | shared | [SPEC-FR-M2.2](functional/M2/SPEC-FR-M2.2.md) |
| SPEC-FR-M2.3 | Keeper Hello World | IMPLEMENTED | keeper | [SPEC-FR-M2.3](functional/M2/SPEC-FR-M2.3.md) |
| SPEC-FR-M2.4 | Agent Hello World | IMPLEMENTED | agent | [SPEC-FR-M2.4](functional/M2/SPEC-FR-M2.4.md) |
| SPEC-FR-M2.5 | Operator Hello World | IMPLEMENTED | operator | [SPEC-FR-M2.5](functional/M2/SPEC-FR-M2.5.md) |
| SPEC-FR-M2.6 | BFF Hello World | IMPLEMENTED | bff | [SPEC-FR-M2.6](functional/M2/SPEC-FR-M2.6.md) |
| SPEC-FR-M2.7 | Container Images (distroless, multi-stage) | IMPLEMENTED | build | [SPEC-FR-M2.7](functional/M2/SPEC-FR-M2.7.md) |
| SPEC-FR-M2.8 | Continuous Integration (GitHub Actions) | ACCEPTED | build | [SPEC-FR-M2.8](functional/M2/SPEC-FR-M2.8.md) |
| SPEC-FR-M2.9 | Project Documentation | IMPLEMENTED | docs | [SPEC-FR-M2.9](functional/M2/SPEC-FR-M2.9.md) |
| SPEC-FR-M2.10 | Avoid Bitnami (Leverage Free & Non-Commercial Infrastructural Dependencies) | IMPLEMENTED | deploy | [SPEC-FR-M2.10](functional/M2/SPEC-FR-M2.10.md) |
| SPEC-FR-M2.11 | Secured Infrastructure Provisioning (Initial Provisioning & TLS Enforcement) | IMPLEMENTED | deploy | [SPEC-FR-M2.11](functional/M2/SPEC-FR-M2.11.md) |

### M3: Keeper Core

| ID | Title | Status | Component | File |
|----|-------|--------|-----------|------|
| SPEC-FR-M3.1 | LLM Provider Bindings & CRUD API | IMPLEMENTED | keeper | [SPEC-FR-M3.1](functional/M3/SPEC-FR-M3.1.md) |
| SPEC-FR-M3.2 | MCP Servers & CRUD API | IMPLEMENTED | keeper | [SPEC-FR-M3.2](functional/M3/SPEC-FR-M3.2.md) |
| SPEC-FR-M3.3 | Skill Collections & CRUD API | IMPLEMENTED | keeper | [SPEC-FR-M3.3](functional/M3/SPEC-FR-M3.3.md) |
| SPEC-FR-M3.4 | Prompt Collections & CRUD API | IMPLEMENTED | keeper | [SPEC-FR-M3.4](functional/M3/SPEC-FR-M3.4.md) |
| SPEC-FR-M3.5 | Agent Domain Model & CRUD API | IMPLEMENTED | keeper | [SPEC-FR-M3.5](functional/M3/SPEC-FR-M3.5.md) |
| SPEC-FR-M3.6 | Community Domain Model & CRUD API | IMPLEMENTED | keeper | [SPEC-FR-M3.6](functional/M3/SPEC-FR-M3.6.md) |
| SPEC-FR-M3.7 | Agent-Community Assignment | IMPLEMENTED | keeper | [SPEC-FR-M3.7](functional/M3/SPEC-FR-M3.7.md) |
| SPEC-FR-M3.8 | PostgreSQL Persistence & Migrations | ACCEPTED | keeper | [SPEC-FR-M3.8](functional/M3/SPEC-FR-M3.8.md) |

### M3: Bugs

| ID | Title | Status | Severity | File |
|----|-------|--------|----------|------|
| BUG-M3.1 | Supporting Entity Models Lack Tenant Segregation | CLOSED | HIGH | [BUG-M3.1](tasks/M3.BUG1/BUG-M3.1.md) |
| BUG-M3.2 | Silent Route Registration Failure due to PostgreSQL Coupling in Keeper Bootstrap | CLOSED | HIGH | [BUG-M3.2](tasks/M3.BUG2/BUG-M3.2.md) |
| BUG-M3.3 | Hexagonal Architecture Violations (Missing Application Service Layer & Flat Bounded Contexts) | OPEN | HIGH | [BUG-M3.3](tasks/M3.BUG3/BUG-M3.3.md) |
| BUG-M3.4 | Broken Observability Context Propagation & Domain Metric Gaps | OPEN | HIGH | [BUG-M3.4](tasks/M3.BUG4/BUG-M3.4.md) |
| BUG-M3.5 | Missing OpenAPI Contract Tests | OPEN | MEDIUM | [BUG-M3.5](tasks/M3.BUG5/BUG-M3.5.md) |
| BUG-M3.6 | Synchronous Blocking Side-Effects in Agent-Community Assignment | OPEN | HIGH | [BUG-M3.6](tasks/M3.BUG6/BUG-M3.6.md) |
| BUG-M3.7 | Health Probes Missing NATS and Redis Dependency Checks | OPEN | MEDIUM | [BUG-M3.7](tasks/M3.BUG7/BUG-M3.7.md) |
| BUG-M3.8 | Stack Dependencies & Migration Framework Deviations | OPEN | MEDIUM | [BUG-M3.8](tasks/M3.BUG8/BUG-M3.8.md) |
| BUG-M3.9 | Misaligned Environment Variable Bindings for Keeper Deployment in Helm Chart | CLOSED | HIGH | [BUG-M3.9](tasks/M3.BUG9/BUG-M3.9.md) |

### M4: Operator Core

| ID | Title | Status | Component | File |
|----|-------|--------|-----------|------|
| SPEC-FR-M4.1 | Agent CRD Definition & Registration | DRAFT | operator | [SPEC-FR-M4.1](functional/M4/SPEC-FR-M4.1.md) |
| SPEC-FR-M4.2 | AgentCommunity CRD Definition | DRAFT | operator | [SPEC-FR-M4.2](functional/M4/SPEC-FR-M4.2.md) |
| SPEC-FR-M4.3 | Reconciliation Controller | DRAFT | operator | [SPEC-FR-M4.3](functional/M4/SPEC-FR-M4.3.md) |
| SPEC-FR-M4.4 | Zero-Scaling Support | DRAFT | operator | [SPEC-FR-M4.4](functional/M4/SPEC-FR-M4.4.md) |
| SPEC-FR-M4.5 | OIDC/JWT Authentication | DRAFT | keeper, shared | [SPEC-FR-M4.5](functional/M4/SPEC-FR-M4.5.md) |
| SPEC-FR-M4.6 | Agent CRD Submission | DRAFT | keeper | [SPEC-FR-M4.6](functional/M4/SPEC-FR-M4.6.md) |

### M5: Agent Core

| ID | Title | Status | Component | File |
|----|-------|--------|-----------|------|
| SPEC-FR-M5.1 | Agent Configuration from CRD Spec | DRAFT | agent | [SPEC-FR-M5.1](functional/M5/SPEC-FR-M5.1.md) |
| SPEC-FR-M5.2 | LLM Reasoning (Brain Adapter) | DRAFT | agent | [SPEC-FR-M5.2](functional/M5/SPEC-FR-M5.2.md) |
| SPEC-FR-M5.3 | Short-Term Memory (Redis) | DRAFT | agent | [SPEC-FR-M5.3](functional/M5/SPEC-FR-M5.3.md) |
| SPEC-FR-M5.4 | Long-Term Memory (Qdrant) | DRAFT | agent | [SPEC-FR-M5.4](functional/M5/SPEC-FR-M5.4.md) |
| SPEC-FR-M5.5 | Tool Invocation (MCP) | DRAFT | agent | [SPEC-FR-M5.5](functional/M5/SPEC-FR-M5.5.md) |
| SPEC-FR-M5.6 | Object Storage (S3/MinIO) | DRAFT | agent | [SPEC-FR-M5.6](functional/M5/SPEC-FR-M5.6.md) |

### M6: Communities & Messaging

| ID | Title | Status | Component | File |
|----|-------|--------|-----------|------|
| SPEC-FR-M6.1 | Community Topology (Hub-Spoke) | DRAFT | keeper, agent | [SPEC-FR-M6.1](functional/M6/SPEC-FR-M6.1.md) |
| SPEC-FR-M6.2 | NATS Inter-Agent Messaging | DRAFT | agent | [SPEC-FR-M6.2](functional/M6/SPEC-FR-M6.2.md) |
| SPEC-FR-M6.3 | NATS Subject Namespacing | DRAFT | agent, keeper | [SPEC-FR-M6.3](functional/M6/SPEC-FR-M6.3.md) |
| SPEC-FR-M6.4 | Thread Management | DRAFT | keeper | [SPEC-FR-M6.4](functional/M6/SPEC-FR-M6.4.md) |
| SPEC-FR-M6.5 | A2A Agent Cards | DRAFT | agent | [SPEC-FR-M6.5](functional/M6/SPEC-FR-M6.5.md) |
| SPEC-FR-M6.6 | Conversation Handoff | DRAFT | agent | [SPEC-FR-M6.6](functional/M6/SPEC-FR-M6.6.md) |
| SPEC-FR-M6.7 | Specialist Agent Spawn | DRAFT | keeper, agent | [SPEC-FR-M6.7](functional/M6/SPEC-FR-M6.7.md) |

### M7: BFF & UIs

| ID | Title | Status | Component | File |
|----|-------|--------|-----------|------|
| SPEC-FR-M7.1 | BFF API Bridge Layer | DRAFT | bff | [SPEC-FR-M7.1](functional/M7/SPEC-FR-M7.1.md) |
| SPEC-FR-M7.2 | Configurator UI | DRAFT | ui | [SPEC-FR-M7.2](functional/M7/SPEC-FR-M7.2.md) |
| SPEC-FR-M7.3 | Auditor UI | DRAFT | ui | [SPEC-FR-M7.3](functional/M7/SPEC-FR-M7.3.md) |
| SPEC-FR-M7.4 | OIDC Login Flow (Keycloak) | DRAFT | bff | [SPEC-FR-M7.4](functional/M7/SPEC-FR-M7.4.md) |

### M8: Governance

| ID | Title | Status | Component | File |
|----|-------|--------|-----------|------|
| SPEC-FR-M8.1 | RBAC Role Model & Route Protection | DRAFT | keeper, shared | [SPEC-FR-M8.1](functional/M8/SPEC-FR-M8.1.md) |
| SPEC-FR-M8.2 | Usage Quotas (community + agent) | DRAFT | keeper | [SPEC-FR-M8.2](functional/M8/SPEC-FR-M8.2.md) |
| SPEC-FR-M8.3 | Quota Enforcement (Redis counters) | DRAFT | keeper | [SPEC-FR-M8.3](functional/M8/SPEC-FR-M8.3.md) |
| SPEC-FR-M8.4 | HITL Yield & Callback Flows | DRAFT | agent, keeper | [SPEC-FR-M8.4](functional/M8/SPEC-FR-M8.4.md) |
| SPEC-FR-M8.5 | Audit Trail (events + queries) | DRAFT | keeper | [SPEC-FR-M8.5](functional/M8/SPEC-FR-M8.5.md) |
| SPEC-FR-M8.6 | Prompt Management (CRUD + versioning) | DRAFT | keeper | [SPEC-FR-M8.6](functional/M8/SPEC-FR-M8.6.md) |
| SPEC-FR-M8.7 | Skills Management (CRUD + MCP attach) | DRAFT | keeper | [SPEC-FR-M8.7](functional/M8/SPEC-FR-M8.7.md) |

### M9: Federation & Hardening

| ID | Title | Status | Component | File |
|----|-------|--------|-----------|------|
| SPEC-FR-M9.1 | A2A HTTP Gateway | DRAFT | keeper | [SPEC-FR-M9.1](functional/M9/SPEC-FR-M9.1.md) |
| SPEC-FR-M9.2 | External Agent Registry | DRAFT | keeper | [SPEC-FR-M9.2](functional/M9/SPEC-FR-M9.2.md) |
| SPEC-FR-M9.3 | Prometheus Metrics Integration | DRAFT | all | [SPEC-FR-M9.3](functional/M9/SPEC-FR-M9.3.md) |
| SPEC-FR-M9.4 | OpenAPI Contract Validation | DRAFT | all | [SPEC-FR-M9.4](functional/M9/SPEC-FR-M9.4.md) |
| SPEC-FR-M9.5 | E2E & Benchmark Tests | DRAFT | test | [SPEC-FR-M9.5](functional/M9/SPEC-FR-M9.5.md) |
| SPEC-FR-M9.6 | Production Helm & Hardening | DRAFT | deploy | [SPEC-FR-M9.6](functional/M9/SPEC-FR-M9.6.md) |
| SPEC-FR-M9.7 | K8s NetworkPolicies | DRAFT | operator | [SPEC-FR-M9.7](functional/M9/SPEC-FR-M9.7.md) |

## Milestones

| Milestone | Title | Status | Specs | File |
|-----------|-------|--------|-------|------|
| M1 | Infrastructure Helm Chart | ✔️ IMPLEMENTED | 2 | [M1](milestones/M1-infrastructure.md) |
| M2 | Application Helm Chart & Component Scaffolding | ✔️ IMPLEMENTED | 11 | [M2](milestones/M2-packaging.md) |
| M3 | Keeper Core | ⬜ PLANNED | 8 | [M3](milestones/M3-keeper-core.md) |
| M4 | Operator Core | ⬜ PLANNED | 6 | [M4](milestones/M4-operator-core.md) |
| M5 | Agent Core | ⬜ PLANNED | 6 | [M5](milestones/M5-agent-core.md) |
| M6 | Communities & Messaging | ⬜ PLANNED | 7 | [M6](milestones/M6-communities.md) |
| M7 | BFF & UIs | ⬜ PLANNED | 4 | [M7](milestones/M7-bff-uis.md) |
| M8 | Governance | ⬜ PLANNED | 7 | [M8](milestones/M8-governance.md) |
| M9 | Federation & Hardening | ⬜ PLANNED | 7 | [M9](milestones/M9-federation-hardening.md) |

## Task Files

| Spec | Tasks | Phase | Directory |
|------|-------|-------|-----------|
| SPEC-FR-M1.1 | 5 | RED → GREEN → REFACTOR | [tasks/M1.1/](tasks/M1.1/) |
| SPEC-FR-M1.2 | 3 | RED → GREEN → REFACTOR | [tasks/M1.2/](tasks/M1.2/) |
| SPEC-FR-M2.1 | 4 | RED → GREEN → REFACTOR | [tasks/M2.1/](tasks/M2.1/) |
| SPEC-FR-M2.2 | 3 | RED → GREEN → REFACTOR | [tasks/M2.2/](tasks/M2.2/) |
| SPEC-FR-M2.3 | 3 | RED → GREEN → REFACTOR | [tasks/M2.3/](tasks/M2.3/) |
| SPEC-FR-M2.4 | 3 | RED → GREEN → REFACTOR | [tasks/M2.4/](tasks/M2.4/) |
| SPEC-FR-M2.5 | 3 | RED → GREEN → REFACTOR | [tasks/M2.5/](tasks/M2.5/) |
| SPEC-FR-M2.6 | 3 | RED → GREEN → REFACTOR | [tasks/M2.6/](tasks/M2.6/) |
| SPEC-FR-M2.7 | 3 | RED → GREEN → REFACTOR | [tasks/M2.7/](tasks/M2.7/) |
| SPEC-FR-M2.8 | 3 | RED → GREEN → REFACTOR | [tasks/M2.8/](tasks/M2.8/) |
| SPEC-FR-M2.9 | 4 | RED → GREEN → REFACTOR | [tasks/M2.9/](tasks/M2.9/) |
| SPEC-FR-M2.10 | 3 | RED → GREEN → REFACTOR | [tasks/M2.10/](tasks/M2.10/) |
| SPEC-FR-M2.11 | 3 | RED → GREEN → REFACTOR | [tasks/M2.11/](tasks/M2.11/) |
| SPEC-FR-M3.1 | 2 | Domain/Persistence & HTTP API Boundaries | [tasks/M3.1/](tasks/M3.1/) |
| SPEC-FR-M3.2 | 2 | Domain/Persistence & HTTP API Boundaries | [tasks/M3.2/](tasks/M3.2/) |
| SPEC-FR-M3.3 | 2 | Domain/Persistence & HTTP API Boundaries | [tasks/M3.3/](tasks/M3.3/) |
| SPEC-FR-M3.4 | 2 | Domain/Persistence & HTTP API Boundaries | [tasks/M3.4/](tasks/M3.4/) |
| SPEC-FR-M3.5 | 2 | Domain/Persistence & HTTP API Boundaries | [tasks/M3.5/](tasks/M3.5/) |
| SPEC-FR-M3.6 | 3 | Domain/Persistence & HTTP API Boundaries | [tasks/M3.6/](tasks/M3.6/) |
| SPEC-FR-M3.7 | 3 | Domain, HTTP API Boundaries & Observability | [tasks/M3.7/](tasks/M3.7/) |
| SPEC-FR-M3.8 | 4 | Domain, HTTP API Boundaries, Observability & Helm | [tasks/M3.8/](tasks/M3.8/) |
