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
- Functional (FR): specs/functional/<FR-XX.Y>.md, specs related to the features to be implemented.
- Tasks: specs/tasks/<FR-ID>/<TASK-XX.Y>.md, specs related to the tasks to be achieved. A task is related to a single FR.

### Rules

1. **No code without a spec.** Every feature, adapter, or NFR change requires a spec document.
2. **No code without a task.** Every task is derived from a spec.
3. **No task can be started without review and approval** from the spec owner.
4. **No task can be completed without review and approval** from the spec owner.
5. **Specs are atomic.** One spec = one testable capability. No compound specs.
6. **Specs are immutable once accepted.** To change behavior, write a superseding spec.
7. **Organize tasks according to boundaries:** by component, by package, by subdomain.
8. **Do not organize tasks by TDD phases:** TDD phases are operational, not organizational.
9. **Tasks MUST be linked to a single spec:** each task must be linked to a single FR spec.
10. **Dependencies between specs MUST be tracked:** a spec can depend on other specs (FR or NFR). Use `Depends On:` field.
11. **Superseding specs MUST be tracked:** a spec can supersede a previous spec. Use `Supersedes:` field.
12. **NFR specs are the foundation:** they are written once and updated only when necessary.
13. **FR specs are the building blocks:** they are written for each feature and are used to derive tasks.

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
