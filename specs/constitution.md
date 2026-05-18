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
| P1 | **Spec-Driven Development** | Every feature MUST have a spec in `specs/` BEFORE code is written. |
| P2 | **TDD (Red → Green → Refactor)** | Tests are written FIRST. Implementation follows. No exception. |
| P3 | **API-First** | All functionality accessible via authenticated REST APIs. UIs are optional consumers. |
| P4 | **Contract-Based** | Components interact through versioned OpenAPI contracts. Independent release cycles. |
| P5 | **Community-Centric** | Agents form communities with configurable topologies and isolation. |
| P6 | **Immutable Specs** | Once a spec is verified, accepted and implemented, it is FROZEN. Changes require a new spec version. |

## 3. Spec-Driven Development Workflow

```
┌──────────┐    ┌──────────────┐    ┌──────────┐    ┌──────────────┐    ┌──────────┐
│ 1. SPEC  │───▸│ 2. REVIEW    │───▸│ 3. RED   │───▸│ 4. GREEN     │───▸│ 5. REFAC │
│ Write    │    │    ACCEPT    │    │ Tests    │    │ Implement    │    │   TOR    │
└──────────┘    └──────────────┘    └──────────┘    └──────────────┘    └──────────┘
     │                │                   │               │                   │
     ▼                ▼                   ▼               ▼                   ▼
  DRAFT            ACCEPTED          IN_PROGRESS       IMPLEMENTED            VERIFIED
  status           status              status              status               status
  in header         in header           in header           in header            in header
```

### Specs catalog

- Non functional (NFR): specs/nonfunctional/, once verified, no need to update them, except for major changes. Prescribe the architecture, technologies, patterns to follow.
- Milestones: specs/milestones/, specs related to the objective of the milestone. They are used to group related features. Once achieved, the milestone is frozen.
- Functional (FR): specs/functional/<milestoneId>/<FR-XX.Y>.md, specs related to the features to be implemented. A FR cannot be related to multiple milestones.
- Tasks: specs/tasks/<FR-ID>/<TASK-XX.Y>.md, specs related to the tasks to be achieved. A task is related to a single FR.

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
├── INDEX.md
├── brainstorming.md
├── constitution.md
├── functional/
│   └── M1/
│       ├── SPEC-FR-M1.1.md
│       ├── SPEC-FR-M1.2.md
│       ├── SPEC-FR-M1.3.md
│       ├── SPEC-FR-M1.4.md
│       └── SPEC-FR-M1.5.md
├── milestones/
│   ├── M1-foundation.md
│   ├── M2-deployable-keeper.md
│   ├── M3-deployable-core.md
│   ├── M4-prompt-skills.md
│   ├── M5-communities.md
│   ├── M6-policies.md
│   ├── M7-operator.md
│   ├── M8-uis-bff.md
│   └── M9-federation-hardening.md
├── nonfunctional/
│   ├── SPEC-NFR-CACHE.md
│   ├── SPEC-NFR-CLOUD.md
│   ├── SPEC-NFR-HEALTH.md
│   ├── SPEC-NFR-HEXAGONAL.md
│   ├── SPEC-NFR-HPA.md
│   ├── SPEC-NFR-HTTP.md
│   ├── SPEC-NFR-LOG.md
│   ├── SPEC-NFR-METRICS.md
│   ├── SPEC-NFR-OPENAPI.md
│   ├── SPEC-NFR-REACTIVE.md
│   └── SPEC-NFR-STACK.md
└── tasks/
    ├── FR-M1.1/
    │   ├── TASK-FR-M1.1-001.md
    │   └── TASK-FR-M1.1-002.md
    ├── FR-M1.2/
    │   └── TASK-FR-M1.2-001.md
    ├── FR-M1.3/
    │   ├── TASK-FR-M1.3-001.md
    │   └── TASK-FR-M1.3-002.md
    ├── FR-M1.4/
    │   ├── TASK-FR-M1.4-001.md
    │   └── TASK-FR-M1.4-002.md
    └── FR-M1.5/
        ├── TASK-FR-M1.5-001.md
        └── TASK-FR-M1.5-002.md
```

## 6. Milestone Registry

| Milestone | Name | Goal | Status |
|-----------|------|------|--------|
| M1 | Foundation | Scaffolding of the project | 🔄 In Progress |
| M2 | Memory, Tools, Storage & Cache | Redis, Qdrant, MCP, S3, Cache adapters | ⬜ Planned |
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
