# Tacito Square — Spec Index

> Single source of truth for all specs. Generated from `specs/` directory.

## Architecture Specs

| ID | Title | Status | File |
|----|-------|--------|------|
| SPEC-ARCH-001 | Hexagonal Architecture | VERIFIED | [SPEC-ARCH-001](architecture/SPEC-ARCH-001-hexagonal.md) |
| SPEC-ARCH-002 | Keeper Data Model | ACCEPTED | [SPEC-ARCH-002](architecture/SPEC-ARCH-002-data-model.md) |
| SPEC-ARCH-003 | Technology Stack | VERIFIED | [SPEC-ARCH-003](architecture/SPEC-ARCH-003-tech-stack.md) |

## Non-Functional Requirement Specs

| ID | Title | Status | Milestone | File |
|----|-------|--------|-----------|------|
| SPEC-NFR-LOG | Structured Logging (zerolog) | VERIFIED | M1 | [SPEC-NFR-LOG](nonfunctional/SPEC-NFR-LOG.md) |
| SPEC-NFR-HTTP | HTTP Framework (Gin) | VERIFIED | M1 | [SPEC-NFR-HTTP](nonfunctional/SPEC-NFR-HTTP.md) |
| SPEC-NFR-HEALTH | Dependency-Aware Health Probes | VERIFIED | M1 | [SPEC-NFR-HEALTH](nonfunctional/SPEC-NFR-HEALTH.md) |
| SPEC-NFR-METRICS | Prometheus Metrics Endpoints | DRAFT | M8 | [SPEC-NFR-METRICS](nonfunctional/SPEC-NFR-METRICS.md) |
| SPEC-NFR-OPENAPI | Live OpenAPI Spec Endpoints | DRAFT | M8 | [SPEC-NFR-OPENAPI](nonfunctional/SPEC-NFR-OPENAPI.md) |
| SPEC-NFR-HPA | Horizontal Pod Autoscaling | DRAFT | M5 | [SPEC-NFR-HPA](nonfunctional/SPEC-NFR-HPA.md) |
| SPEC-NFR-CACHE | Redis Infrastructure Cache | IMPLEMENTED | M2 | [SPEC-NFR-CACHE](nonfunctional/SPEC-NFR-CACHE.md) |

## Functional Requirement Specs

### FR-01: Agent Lifecycle Management

| ID | Title | Status | Milestone | File |
|----|-------|--------|-----------|------|
| SPEC-FR-01.1 | Spawn agent | VERIFIED | M1, M3 | [SPEC-FR-01.1](functional/FR-01/SPEC-FR-01.1.md) |
| SPEC-FR-01.2 | State transitions | VERIFIED | M1 | [SPEC-FR-01.2](functional/FR-01/SPEC-FR-01.2.md) |
| SPEC-FR-01.3 | Config snapshot | VERIFIED | M1 | [SPEC-FR-01.3](functional/FR-01/SPEC-FR-01.3.md) |
| SPEC-FR-01.4 | Heartbeat processing | DRAFT | M1 | — |
| SPEC-FR-01.5 | Terminate agents | VERIFIED | M1 | [SPEC-FR-01.5](functional/FR-01/SPEC-FR-01.5.md) |
| SPEC-FR-01.6 | Audit log per transition | DRAFT | M4 | — |

### FR-02: Prompt Management

| ID | Title | Status | Milestone |
|----|-------|--------|-----------|
| SPEC-FR-02.1 | Prompt CRUD | DRAFT | M4 |
| SPEC-FR-02.2 | Prompt versioning | DRAFT | M4 |
| SPEC-FR-02.3 | Prompt import/export | DRAFT | M4 |

### FR-03: Platform Foundations

| ID | Title | Status | Milestone | File |
|----|-------|--------|-----------|------|
| SPEC-FR-03.1 | PostgreSQL AgentStore and Migrations | DRAFT | M3 | [SPEC-FR-03.1](functional/FR-03/SPEC-FR-03.1.md) |
| SPEC-FR-03.2 | Gin RBAC middleware | DRAFT | M3 | [SPEC-FR-03.2](functional/FR-03/SPEC-FR-03.2.md) |

### FR-17: Skills Management

| ID | Title | Status | Milestone |
|----|-------|--------|-----------|
| SPEC-FR-17.1 | Skill CRUD | DRAFT | M4 |
| SPEC-FR-17.2 | MCP tool attach/detach | DRAFT | M4 |
| SPEC-FR-17.3 | Skill assignment at spawn | DRAFT | M4 |

### FR-04: Agent Reasoning & Conversation

| ID | Title | Status | Milestone | File |
|----|-------|--------|-----------|------|
| SPEC-FR-04.1 | LLM reasoning (brain adapter) | VERIFIED | M1 | [SPEC-FR-04.1](functional/FR-04/SPEC-FR-04.1.md) |
| SPEC-FR-04.2 | Short-term memory (Redis) | IMPLEMENTED | M2 | [SPEC-FR-04.2](functional/FR-04/SPEC-FR-04.2.md) |
| SPEC-FR-04.3 | Long-term memory (Qdrant) | IMPLEMENTED | M2 | [SPEC-FR-04.3](functional/FR-04/SPEC-FR-04.3.md) |
| SPEC-FR-04.4 | Tool invocation (MCP) | IMPLEMENTED | M2 | [SPEC-FR-04.4](functional/FR-04/SPEC-FR-04.4.md) |
| SPEC-FR-04.5 | Specialist agent spawn | DRAFT | M5 | — |
| SPEC-FR-04.6 | Conversation handoff | DRAFT | M5 | — |
| SPEC-FR-04.7 | HITL yield | DRAFT | M6 | — |

### FR-05: Community Management

| ID | Title | Status | Milestone |
|----|-------|--------|-----------|
| SPEC-FR-05.1 | Community domain & service | DRAFT | M5 |
| SPEC-FR-05.2 | Hub-Spoke topology | DRAFT | M5 |
| SPEC-FR-05.3 | NATS subject namespacing | DRAFT | M5 |
| SPEC-FR-05.4 | K8s NetworkPolicies | DRAFT | M6 |
| SPEC-FR-05.5 | Multi-thread engagements | DRAFT | M5 |
| SPEC-FR-05.6 | Thread CRUD | DRAFT | M5 |

### FR-06: Inter-Agent Communication

| ID | Title | Status | Milestone |
|----|-------|--------|-----------|
| SPEC-FR-06.1 | A2A Agent Cards | DRAFT | M5 |
| SPEC-FR-06.2 | NATS internal messaging | DRAFT | M5 |
| SPEC-FR-06.3 | A2A HTTP gateway | DRAFT | M9 |
| SPEC-FR-06.4 | Hub routing | DRAFT | M5 |
| SPEC-FR-06.5 | External source registry | DRAFT | M9 |
| SPEC-FR-06.6 | External agent messaging | DRAFT | M9 |
| SPEC-FR-06.7 | External source health | DRAFT | M9 |

### FR-07: K8s Operator

| ID | Title | Status | Milestone |
|----|-------|--------|-----------|
| SPEC-FR-07.1 | Agent CRD | DRAFT | M7 |
| SPEC-FR-07.2 | AgentCommunity CRD | DRAFT | M7 |
| SPEC-FR-07.3 | Validating webhooks | DRAFT | M7 |
| SPEC-FR-07.4 | Mutating webhooks | DRAFT | M7 |

### FR-08: APIs, UIs & Authentication

| ID | Title | Status | Milestone | File |
|----|-------|--------|-----------|------|
| SPEC-FR-08.1 | Keeper REST API | VERIFIED | M1 | [SPEC-FR-08.1](functional/FR-08/SPEC-FR-08.1.md) |
| SPEC-FR-08.2 | User REST API | DRAFT | M8 | — |
| SPEC-FR-08.3 | Keeper UI | DRAFT | M8 | — |
| SPEC-FR-08.4 | User UI | DRAFT | M8 | — |
| SPEC-FR-08.5 | OIDC/JWT auth | VERIFIED | M1, M8 | [SPEC-FR-08.5](functional/FR-08/SPEC-FR-08.5.md) |
| SPEC-FR-08.6 | BFF layer | DRAFT | M8 | — |

### FR-09: Observability & Traceability

| ID | Title | Status | Milestone | File |
|----|-------|--------|-----------|------|
| SPEC-FR-09.1 | OpenTelemetry tracing | VERIFIED | M1 | [SPEC-FR-09.1](functional/FR-09/SPEC-FR-09.1.md) |
| SPEC-FR-09.2 | Structured logging (zerolog) | VERIFIED | M1 | (see NFR-LOG) |
| SPEC-FR-09.3 | Prometheus metrics | DRAFT | M9 | — |
| SPEC-FR-09.4 | Audit log queries | DRAFT | M5 | — |
| SPEC-FR-09.5–09.9 | Health probes | VERIFIED | M1 | (see NFR-HEALTH) |

### FR-10: Testing & Quality

| ID | Title | Status | Milestone | File |
|----|-------|--------|-----------|------|
| SPEC-FR-10.1 | Unit tests (TDD) | VERIFIED | M1+ | [SPEC-FR-10.1](functional/FR-10/SPEC-FR-10.1.md) |
| SPEC-FR-10.2 | Integration tests | DRAFT | M2+ | — |
| SPEC-FR-10.3 | Operator tests | DRAFT | M7 | — |
| SPEC-FR-10.4 | E2E tests | DRAFT | M9 | — |
| SPEC-FR-10.5 | Benchmark tests | DRAFT | M9 | — |
| SPEC-FR-10.6 | Concurrency tests | DRAFT | M9 | — |
| SPEC-FR-10.7 | Makefile targets | VERIFIED | M1 | [SPEC-FR-10.7](functional/FR-10/SPEC-FR-10.7.md) |

### FR-11: Human in the Loop (HITL)

| ID | Title | Status | Milestone |
|----|-------|--------|-----------|
| SPEC-FR-11.1 | HITL Agent Card flag | DRAFT | M6 |
| SPEC-FR-11.2 | HITL yield in reasoning | DRAFT | M6 |
| SPEC-FR-11.3 | HITL callback persistence | DRAFT | M6 |
| SPEC-FR-11.4 | HITL human response | DRAFT | M6 |
| SPEC-FR-11.5 | HITL TTL/escalation | DRAFT | M6 |
| SPEC-FR-11.6 | HITL audit events | DRAFT | M6 |

### FR-12: API-First & Component Versioning

| ID | Title | Status | Milestone | File |
|----|-------|--------|-----------|------|
| SPEC-FR-12.1 | Bearer JWT auth | VERIFIED | M1 | [SPEC-FR-12.1](functional/FR-12/SPEC-FR-12.1.md) |
| SPEC-FR-12.2 | API-first design | VERIFIED | M1 | [SPEC-FR-12.2](functional/FR-12/SPEC-FR-12.2.md) |
| SPEC-FR-12.3 | Independent versioning | VERIFIED | M1 | [SPEC-FR-12.3](functional/FR-12/SPEC-FR-12.3.md) |
| SPEC-FR-12.4 | OpenAPI contracts | DRAFT | M9 | — |
| SPEC-FR-12.5 | Contract tests | DRAFT | M9 | — |
| SPEC-FR-12.6 | Helm sub-charts | VERIFIED | M1 | [SPEC-FR-12.6](functional/FR-12/SPEC-FR-12.6.md) |

### FR-13: Accountability & RBAC

| ID | Title | Status | Milestone | File |
|----|-------|--------|-----------|------|
| SPEC-FR-13.1 | RBAC role model | DRAFT | M3 | — |
| SPEC-FR-13.3 | Principal logging | DRAFT | M3 | — |
| SPEC-FR-13.4 | Audit trail with actor identity | DRAFT | M5, M3 | — |
| SPEC-FR-13.5 | Keycloak realm via Helm | VERIFIED | M1 | [SPEC-FR-13.5](functional/FR-13/SPEC-FR-13.5.md) |
| SPEC-FR-13.6 | Role-based route protection | DRAFT | M3 | — |
| SPEC-FR-13.7 | Service-to-service auth | DRAFT | M8 | — |

### FR-14: Default & Built-in Agents (NEW)

| ID | Title | Status | Milestone | File |
|----|-------|--------|-----------|------|
| SPEC-FR-14.1 | Default community agent (entry point) | DRAFT | M6 | [SPEC-FR-14](functional/FR-14/SPEC-FR-14.md) |
| SPEC-FR-14.2 | Built-in agent archetypes | DRAFT | M6 | [SPEC-FR-14](functional/FR-14/SPEC-FR-14.md) |
| SPEC-FR-14.3 | Community template with agent manifest | DRAFT | M6 | [SPEC-FR-14](functional/FR-14/SPEC-FR-14.md) |

### FR-15: Usage Quotas (NEW)

| ID | Title | Status | Milestone | File |
|----|-------|--------|-----------|------|
| SPEC-FR-15.1 | Community quotas | DRAFT | M6 | [SPEC-FR-15](functional/FR-15/SPEC-FR-15.md) |
| SPEC-FR-15.2 | Agent quotas | DRAFT | M6 | [SPEC-FR-15](functional/FR-15/SPEC-FR-15.md) |
| SPEC-FR-15.3 | Quota enforcement (Redis counters) | DRAFT | M6, M7 | [SPEC-FR-15](functional/FR-15/SPEC-FR-15.md) |
| SPEC-FR-15.4 | Quota tracking & reporting | DRAFT | M6 | [SPEC-FR-15](functional/FR-15/SPEC-FR-15.md) |

### FR-16: Object Storage (NEW)

| ID | Title | Status | Milestone | File |
|----|-------|--------|-----------|------|
| SPEC-FR-16.1 | S3 outbound port (BlobStore) | IMPLEMENTED | M2 | [SPEC-FR-16](functional/FR-16/SPEC-FR-16.md) |
| SPEC-FR-16.2 | S3/MinIO adapter | IMPLEMENTED | M2 | [SPEC-FR-16](functional/FR-16/SPEC-FR-16.md) |
| SPEC-FR-16.3 | Payload offloading (size threshold) | IMPLEMENTED | M2 | [SPEC-FR-16](functional/FR-16/SPEC-FR-16.md) |

## Milestone Specs

| Milestone | Status | File |
|-----------|--------|------|
| M1: Foundation | ✅ COMPLETE | [M1](milestones/M1-foundation.md) |
| M2: Memory, Tools, Storage & Cache | 🔄 IN PROGRESS | [M2](milestones/M2-memory-tools.md) |
| M3: Deployable Core | ⬜ PLANNED | [M3](milestones/M3-deployable-core.md) |
| M4: Prompt & Skills | ⬜ PLANNED | [M4](milestones/M4-prompt-skills.md) |
| M5: Communities & Messaging | ⬜ PLANNED | [M5](milestones/M5-communities.md) |
| M6: Policies & Governance | ⬜ PLANNED | [M6](milestones/M6-policies.md) |
| M7: K8s Operator | ⬜ PLANNED | [M7](milestones/M7-operator.md) |
| M8: UIs & BFF | ⬜ PLANNED | [M8](milestones/M8-uis-bff.md) |
| M9: Federation & Hardening | ⬜ PLANNED | [M9](milestones/M9-federation-hardening.md) |

## Task Files

| Milestone | Status | Tasks | Directory |
|-----------|--------|-------|-----------|
| M1: Foundation | ✅ COMPLETE | 53/53 | [tasks/M1/](tasks/M1/) |
| M2: Memory, Tools, Storage & Cache | 🔄 IN PROGRESS | 20/25 | [tasks/M2/](tasks/M2/) |
