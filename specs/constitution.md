# Tacito Square — Project Constitution

> The governing document for Tacito Square. Every design decision, every line of code,
> every deployment artifact traces back to a spec referenced by this constitution.

## 1. Mission

Build **Tacito Square**, a Kubernetes-native multi-agent system where autonomous AI agents
form communities, reason with LLMs, use tools, and coordinate through structured protocols —
all while remaining accountable, observable, and scalable.

## 2. Governing Principles

| # | Principle | Enforcement |
|---|-----------|-------------|
| P1 | **Hexagonal Architecture** | Domain logic MUST NOT import adapters. Ports define contracts. |
| P2 | **Stateless Agents** | Agent containers receive ALL configuration via ConfigMap/Secret at deploy time. |
| P3 | **Spec-Driven Development** | Every feature MUST have a spec in `specs/` BEFORE code is written. |
| P4 | **TDD (Red → Green → Refactor)** | Tests are written FIRST. Implementation follows. No exception. |
| P5 | **API-First** | All functionality accessible via authenticated REST APIs. UIs are optional consumers. |
| P6 | **Contract-Based** | Components interact through versioned OpenAPI contracts. Independent release cycles. |
| P7 | **Community-Centric** | Agents form communities with configurable topologies and isolation. |
| P8 | **Accountable** | RBAC enforced at API layer. All mutations audited with actor identity. |
| P9 | **Observable** | Distributed tracing (OTel), structured logging (zerolog), health probes. |
| P10 | **Immutable Specs** | Once a spec is accepted and implemented, it is FROZEN. Changes require a new spec version. |

## 3. Spec-Driven Development Workflow

```
┌──────────┐    ┌──────────────┐    ┌──────────┐    ┌──────────────┐    ┌──────────┐
│ 1. SPEC  │───▸│ 2. REVIEW &  │───▸│ 3. RED   │───▸│ 4. GREEN     │───▸│ 5. REFAC │
│ Write    │    │    ACCEPT     │    │ Tests    │    │ Implement    │    │   TOR    │
└──────────┘    └──────────────┘    └──────────┘    └──────────────┘    └──────────┘
     │                │                   │               │                   │
     ▼                ▼                   ▼               ▼                   ▼
  specs/FR-XX.md   ACCEPTED           *_test.go        *.go              Optimize
  specs/NFR-XX.md  status in          (FAILS)          (PASSES)          (tests
                   spec header                                           still pass)
```

### Rules

1. **No code without a spec.** Every feature, adapter, or NFR change requires a spec document.
2. **Specs are atomic.** One spec = one testable capability. No compound specs.
3. **Specs are immutable once accepted.** To change behavior, write a superseding spec.
4. **Tasks are derived from specs.** Each spec generates one or more atomic tasks.
5. **Status flows one way**: `DRAFT → ACCEPTED → IN_PROGRESS → IMPLEMENTED → VERIFIED`.

## 4. Spec Document Format

Every spec file MUST follow this structure:

```markdown
# SPEC-{ID}: {Title}

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-{FR|NFR}-{XX.Y}                       |
| Status        | DRAFT | ACCEPTED | IN_PROGRESS | IMPLEMENTED | VERIFIED |
| Milestone     | M{N}                                        |
| FR/NFR Ref    | FR-{XX.Y} or NFR-{ID}                      |
| Component     | agent | keeper | operator | bff | shared    |
| Depends On    | SPEC-{ID}, ...                              |
| Supersedes    | SPEC-{ID} (if replacing a previous spec)    |

## Context
Why this spec exists. Problem statement.

## Specification
Precise, testable requirements. Use MUST, SHOULD, MAY (RFC 2119).

## Acceptance Criteria
Numbered list of verifiable conditions.

## Test Plan
Exact test cases (unit, integration, benchmark as applicable).

## API Contract (if applicable)
Request/response shapes, status codes.

## Files Affected
List of files to create or modify.
```

## 5. Directory Structure

```
specs/
├── constitution.md              ← This file
├── architecture/
│   ├── SPEC-ARCH-001-hexagonal.md
│   ├── SPEC-ARCH-002-data-model.md
│   └── SPEC-ARCH-003-tech-stack.md
├── functional/
│   ├── FR-01/                   ← Agent Lifecycle Management
│   │   ├── SPEC-FR-01.1.md
│   │   ├── SPEC-FR-01.2.md
│   │   └── ...
│   ├── FR-02/                   ← Prompt Management
│   ├── FR-03/                   ← Skills Management
│   ├── FR-04/                   ← Agent Reasoning
│   ├── FR-05/                   ← Community Management
│   ├── FR-06/                   ← Inter-Agent Communication
│   ├── FR-07/                   ← K8s Operator
│   ├── FR-08/                   ← APIs, UIs & Authentication
│   ├── FR-09/                   ← Observability
│   ├── FR-10/                   ← Testing & Quality
│   ├── FR-11/                   ← HITL
│   ├── FR-12/                   ← API-First & Versioning
│   ├── FR-13/                   ← Accountability & RBAC
│   ├── FR-14/                   ← Default & Built-in Agents
│   ├── FR-15/                   ← Usage Quotas
│   └── FR-16/                   ← Object Storage (S3)
├── nonfunctional/
│   ├── SPEC-NFR-LOG.md
│   ├── SPEC-NFR-HTTP.md
│   ├── SPEC-NFR-HEALTH.md
│   ├── SPEC-NFR-METRICS.md
│   ├── SPEC-NFR-OPENAPI.md
│   ├── SPEC-NFR-HPA.md
│   └── SPEC-NFR-CACHE.md
├── milestones/
│   ├── M1-foundation.md
│   ├── M2-memory-tools.md
│   ├── M3-deployable-core.md
│   ├── M4-prompt-skills.md
│   ├── M5-communities.md
│   ├── M6-policies.md
│   ├── M7-operator.md
│   ├── M8-uis-bff.md
│   └── M9-federation-hardening.md
└── tasks/
    ├── M1/
    │   ├── TASK-M1-001.md
    │   ├── TASK-M1-002.md
    │   └── ...
    ├── M2/
    └── ...
```

## 6. Milestone Registry

| Milestone | Name | Goal | Status |
|-----------|------|------|--------|
| M1 | Foundation | Walking skeleton: Agent + Keeper + Helm | ✅ Complete |
| M2 | Memory, Tools, Storage & Cache | Redis, Qdrant, MCP, S3, Cache adapters | 🔄 In Progress |
| M3 | Deployable Core | PG persistence, RBAC, production Helm | ⬜ Planned |
| M4 | Prompt & Skills | Versioned prompts, skill CRUD, spawn integration | ⬜ Planned |
| M5 | Communities & Messaging | Threads, Hub-Spoke, NATS, A2A Cards, audit | ⬜ Planned |
| M6 | Policies & Governance | HITL, quotas, default agents, NetworkPolicies | ⬜ Planned |
| M7 | K8s Operator | CRDs, reconcilers, webhooks, HPA | ⬜ Planned |
| M8 | UIs & BFF | React 19, OIDC, Gin BFF | ⬜ Planned |
| M9 | Federation & Hardening | A2A gateway, metrics, OpenAPI, CI/CD, E2E, benchmarks | ⬜ Planned |

## 7. Component Registry

| Component | Module Path | Version File | Port | Depends On |
|-----------|-------------|--------------|------|------------|
| Agent | `internal/agent` | `VERSION.agent` | 8090 | NATS, Redis, Qdrant, LLM, MCP |
| Keeper | `internal/keeper` | `VERSION.keeper` | 8080 | PostgreSQL, NATS, Keycloak |
| Operator | `operator/` | `VERSION.operator` | — | K8s API |
| BFF | `internal/bff` | `VERSION.bff` | 8081 | Keeper API, Keycloak |

## 8. Technology Decisions (Locked)

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Language | Go 1.26 | Fast compile, sub-second bootstrap, easy K8s integration |
| HTTP Framework | Gin | Struct binding, middleware chain, high performance |
| Logging | zerolog | Zero-alloc JSON, trace_id + claims enrichment |
| Tracing | OpenTelemetry OTLP gRPC | Vendor-neutral, W3C TraceContext propagation |
| Messaging | NATS | Lightweight, subject-based routing, thread-scoped |
| STM | Redis | TTL, key namespacing per thread |
| LTM | Qdrant | Vector search for long-term semantic memory |
| Persistence | PostgreSQL + pgx + golang-migrate | Keeper state, audit log |
| Testing | testify + testcontainers-go | TDD mocks + real infra for integration |
| IAM | Keycloak (OIDC) | Dev realm provisioned via Helm |
| Deployment | Helm umbrella chart | Single install, dependency toggles |
| Docker base | distroless | Minimal attack surface |
