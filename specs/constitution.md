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

- Non functional (NFR): Codified as Agent Rules under `.agents/rules/` (e.g., cloud-first, code architecture, observability, K8s, and stack constraints). They prescribe the architecture, technologies, patterns, and system guidelines to follow.
- Functional (FR): specs/functional/<FR-XX.Y>.md, specs related to the features to be implemented.
- Tasks: specs/tasks/<FR-ID>/<TASK-XX.Y>.md, specs related to the tasks to be achieved. A task is related to a single FR.

### Rules

All development workflows, task management, and testing lifecycles are codified and actively enforced by the Agent Rules. In particular:
- Refer to [.agents/rules/spec-driven-development.md](file:///Users/R.Pasquini/Projects/side/tacito-square/.agents/rules/spec-driven-development.md) for strict Spec-Driven & Test-Driven (TDD) compliance and task organization rules.
- Refer to all other `.agents/rules/*.md` files for architectural, infrastructural, and non-functional rules.


## 4. Spec Document Format

Every spec file MUST follow this structure:

```markdown
# SPEC-{ID}: {Title}

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-{FR|NFR}-{XX.Y}                       |
| Status        | DRAFT | ACCEPTED | IN_PROGRESS | IMPLEMENTED | VERIFIED |
| Milestone     | MX (functional specs only)                  |
| Component     | keeper, agent, operator, bff, shared, deploy, build, ui, test, docs |
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

## 5. Milestone Lifecycle

Milestones aggregate multiple functional specs (FR) representing key deliverables. A milestone’s status transitions automatically based on the status of its required specs:

| Status | Description | Transition Rule |
|--------|-------------|-----------------|
| `⬜ PLANNED` | The milestone is defined and planned. | All required specs are in `DRAFT` or `ACCEPTED` status. |
| `🏃 IN_PROGRESS` | Active execution. | At least one required spec is `IN_PROGRESS` or `IMPLEMENTED`, but not all are implemented. |
| `✔️ IMPLEMENTED` | All specs are implemented. | Every required spec has achieved at least `IMPLEMENTED` status. |
| `🎉 COMPLETED` | Fully verified. | Every required spec has achieved `VERIFIED` status. |
