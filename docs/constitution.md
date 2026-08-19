# Tacito Square — Project Constitution

> The governing document for Tacito Square. Every design decision, every line of code,
> every deployment artifact traces back to a specification referenced by this constitution.

## 1. Mission

Build **Tacito Square**, a Kubernetes-native multi-agent system where autonomous AI agents
form communities, reason with LLMs, use tools, and coordinate through structured protocols —
all while remaining accountable, observable, and scalable.

## 2. Governing Principles

| # | Principle | Enforcement |
|---|-----------|-------------|
| P1 | **Spec-Driven Development** | Every feature MUST have an accepted GitHub Issue (`type:spec`) BEFORE code is written. |
| P2 | **TDD (Red → Green → Refactor)** | Tests are written FIRST on feature branches. Implementation follows. PRs reference `Fixes #<id>`. |
| P3 | **API-First** | All functionality accessible via authenticated REST APIs. UIs are optional consumers. |
| P4 | **Contract-Based** | Components interact through versioned OpenAPI contracts. Independent release cycles. |
| P5 | **Community-Centric** | Agents form communities with configurable topologies and isolation. |
| P6 | **Immutable Specs** | Once a spec issue is verified and closed, it is FROZEN. Behavior changes require a new superseding issue. |

## 3. Spec-Driven Development Workflow

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ 1. SPEC ISSUE   │───▸│ 2. REVIEW &     │───▸│ 3. BRANCH &     │───▸│ 4. PR CREATION  │───▸│ 5. REVIEW &     │
│ Create Issue    │    │    ACCEPT       │    │    TDD (R-G-R)  │    │    (Fixes #..)  │    │    MERGE        │
└─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────┘
         │                      │                      │                      │                      │
         ▼                      ▼                      ▼                      ▼                      ▼
   status:draft          status:accepted       status:in-progress     status:implemented     status:verified /
   (GitHub Issue)        (Human Approved)       (Feature Branch)       (Pull Request)         closed on merge
```

### Specs & Governance Catalog

- **Non-Functional Requirements (NFR)**: Codified as Agent Rules under `.agents/rules/` (e.g., cloud-first, code architecture, observability, K8s best practices, and tech stack constraints).
- **Functional Specifications (FR)**: Tracked as GitHub Issues with the label `type:spec` using `.github/ISSUE_TEMPLATE/spec.yml`.
- **Tasks & Work Breakdown**: Tracked as GitHub Sub-Issues / Tasks with label `type:task` using `.github/ISSUE_TEMPLATE/task.yml`.
- **Defects & Bugs**: Tracked as GitHub Issues with label `type:bug` using `.github/ISSUE_TEMPLATE/bug.yml`.
- **Architecture Foundations**: Living system architectural blueprints and domain concepts are maintained under `docs/architecture/`.

### Rules & Development Lifecycles

All development workflows, task management, and testing lifecycles are codified and actively enforced by the Agent Rules:
- Refer to [.agents/rules/spec-driven-development.md](file:///Users/R.Pasquini/Projects/side/tacito-square/.agents/rules/spec-driven-development.md) for strict Spec-Driven & Test-Driven (TDD) compliance, branch workflows, and PR linking.
- Refer to all other `.agents/rules/*.md` files for architectural, infrastructural, and non-functional rules.

## 4. GitHub Issue Specification Format

Every functional specification issue MUST follow the structured form defined in `.github/ISSUE_TEMPLATE/spec.yml`:

- **Context & Problem Statement**: Why this specification exists, user motivation, and architectural background.
- **Primary Component**: `comp:keeper`, `comp:agent`, `comp:operator`, `comp:bff`, `comp:ui`, `comp:shared`, `comp:deploy`.
- **RFC 2119 Specifications**: Precise requirements using `MUST`, `SHOULD`, and `MAY`.
- **Acceptance Criteria**: Numbered list of verifiable conditions for success.
- **Test Plan**: Specific automated test cases (unit, integration with testcontainers, contract, manual).
- **API Contract / Messaging Schema**: Request/response shapes, status codes, or NATS topics.
- **Affected Files & Packages**: Anticipated codebase areas.

## 5. Milestone & Release Lifecycle

Milestones aggregate multiple specification issues representing key deliverables:

| Status | Description | Transition Rule |
|--------|-------------|-----------------|
| `⬜ PLANNED` | Milestone is defined and planned. | All required spec issues are in `status:draft` or `status:accepted`. |
| `🏃 IN_PROGRESS` | Active execution. | At least one spec issue is `status:in-progress` on a feature branch or PR. |
| `✔️ IMPLEMENTED` | All specs are implemented in PRs. | Every required spec has an approved PR or `status:implemented`. |
| `🎉 COMPLETED` | Fully verified and merged. | All spec issues are `status:verified` / `closed` on `main`. |
