# Tacito Square — Specifications Index & Tracking

> As of **Issue #23**, active specification, task breakdown, defect tracking, and milestone management have transitioned from local repository Markdown files to **GitHub Issues, Labels & GitHub Projects**.

---

## Active Development & Tracking (GitHub Native)

| Tracking Concept | Location & Query | Description |
|---|---|---|
| **Functional & Architecture Specs** | [GitHub Issues (`type:spec`)](https://github.com/morphy76/tacito-square/issues?q=is%3Aissue+label%3Atype%3Aspec) | Business context, RFC 2119 specs, acceptance criteria, test plans, and API contracts. |
| **Tasks & Work Breakdown** | [GitHub Issues (`type:task`)](https://github.com/morphy76/tacito-square/issues?q=is%3Aissue+label%3Atype%3Atask) | Subsystem / DDD aggregate tasks with TDD plans (RED, GREEN, REFACTOR). |
| **Bugs & Defects** | [GitHub Issues (`type:bug`)](https://github.com/morphy76/tacito-square/issues?q=is%3Aissue+label%3Atype%3Abug) | Defect reports with repro steps and failing test plans. |
| **Milestones** | [GitHub Milestones](https://github.com/morphy76/tacito-square/milestones) | Aggregate deliverables and release progress tracking. |

---

## Governance & Architecture References

- **Project Constitution**: [specs/constitution.md](constitution.md) (or [docs/constitution.md](../docs/constitution.md))
- **Spec-Driven & TDD Rule**: [.agents/rules/spec-driven-development.md](../.agents/rules/spec-driven-development.md)
- **Architectural Foundations & Domain**: [docs/architecture/overview.md](../docs/architecture/overview.md)
- **E2E & Testing Strategy**: [docs/testing/e2e-test-strategy.md](../docs/testing/e2e-test-strategy.md)
- **Non-Functional Requirements & Agent Rules**: [.agents/rules/](../.agents/rules/)
- **Historical Deliverables Archive (M1–M6)**: [specs/archive/README.md](archive/README.md)

---

## Issue Template Forms

When proposing a new specification, task, or bug fix, use the GitHub Issue Forms:
- [Specification Form](../.github/ISSUE_TEMPLATE/spec.yml)
- [Task Breakdown Form](../.github/ISSUE_TEMPLATE/task.yml)
- [Bug Report Form](../.github/ISSUE_TEMPLATE/bug.yml)
