---
trigger: always_on
globs: ["**/*.md", "**/*.yml", "**/*.yaml"]
description: Collaborative specification and requirements elicitation standards between human and developer agents.
---

# Collaborative Spec Definition & Requirements Elicitation

This rule governs how AI coding agents collaborate with the human owner to define, refine, and accept new specifications (`type:spec`), tasks (`type:task`), and defect reports (`type:bug`) in Tacito Square.

## 1. Collaborative Interview Protocol (Principle P1)

Agents must **never** draft specifications or make major architectural decisions in isolation. Follow this structured interview protocol:

- **Proactive Elicitation**: Conduct an interactive interview to uncover requirements, edge cases, and constraints.
- **Batched / Focused Inquiries**: Ask focused questions one at a time or in small logical groups. Avoid overwhelming the user with massive multi-topic questionnaires.
- **Challenge Ambiguities**: If a requirement is underspecified (e.g. missing HTTP error payloads, undefined concurrency limits, missing tenant scoping), explicitly propose 2-3 options with a recommended default and seek confirmation.
- **Architectural Grounding**: Anchor all discussions in the Tacito Square system architecture ([`docs/architecture/overview.md`](../../docs/architecture/overview.md)) and the Project Constitution ([`specs/constitution.md`](../../specs/constitution.md)).

## 2. RFC 2119 Requirement Rigor

All functional requirements in a specification issue MUST be drafted using RFC 2119 keywords:

- **MUST / SHALL**: Absolute, non-negotiable requirements for system compliance.
- **MUST NOT / SHALL NOT**: Absolute prohibitions.
- **SHOULD / RECOMMENDED**: Valid reasons may exist in particular circumstances to ignore, but the full implications must be understood and carefully weighed.
- **MAY / OPTIONAL**: Truly optional items.

Example:
```markdown
1. The Keeper component MUST reject any agent creation payload where `name` is empty or exceeds 128 characters.
2. The Agent runtime SHOULD gracefully degrade and return cached responses if the LTM Qdrant service is unreachable.
3. The BFF component MAY expose optional GraphQL subscriptions for real-time thread updates.
```

## 3. Issue Taxonomy & Classification

Ensure every work item is classified correctly before drafting:

| Issue Type | Label | Scope & Boundary |
|---|---|---|
| **Functional Spec** | `type:spec` | Defines an atomic business capability, API contract, acceptance criteria, and test plan. |
| **Sub-Task** | `type:task` | Decomposes a spec into a single logical subsystem/package boundary with TDD phases. |
| **Defect / Bug** | `type:bug` | Encapsulates reproduction steps, expected vs. actual behavior, and a failing test plan. |
| **Architectural RFC** | `type:refactor` / RFC | System-wide metamodel, cross-cutting pattern, or major refactoring design document. |

## 4. Spec Acceptance Quality Gate Checklist

An issue **MUST NOT** transition from `status:draft` to `status:accepted` until it satisfies this complete quality gate:

- [ ] **Context & Problem Statement**: Clear business motivation and problem description.
- [ ] **Bounded Context & Component Tag**: Declares primary component (`comp:keeper`, `comp:agent`, `comp:operator`, `comp:bff`, `comp:ui`, `comp:shared`, `comp:deploy`).
- [ ] **RFC 2119 Requirements**: Explicit numbered list of `MUST`/`SHOULD`/`MAY` clauses.
- [ ] **Verifiable Acceptance Criteria**: Numbered list of verifiable conditions for success (no vague goals like "fast" or "clean").
- [ ] **Automated Test Plan**: Concrete test cases for unit tests, integration tests (with testcontainers), and contract tests.
- [ ] **API / Data Contract**: Request/response schemas, HTTP status codes, or NATS message topic payloads.
- [ ] **Multi-Tenancy & Security**: Explicit tenant isolation and authorization handling.
