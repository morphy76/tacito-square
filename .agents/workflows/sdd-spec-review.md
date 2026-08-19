---
trigger: manual
description: Reviewing, refining, and accepting GitHub functional specification issues collaboratively with the user.
---

# SDD Workflow: Spec Review & Refinement (Cooperative Dialogue)

This workflow defines the mandatory process for reviewing, technical refinement, and cooperative discussion of any drafted functional specification GitHub Issue, fully aligned with the [Project Constitution](specs/constitution.md).

---

## Step 1: Pre-Review Compliance Check
When an issue is in `status:draft`, or when the User requests a spec review, the agent MUST evaluate the issue details (`gh issue view <ID>`) against the project guidelines:

1.  **Fidelity to Format**: Confirm that the specification issue contains all standard sections (Context, RFC 2119 Specifications, Acceptance Criteria, Test Plan, API Contract, Files Affected) per `.github/ISSUE_TEMPLATE/spec.yml`.
2.  **Hexagonal DDD Decoupling**: Check that the specification strictly maintains domain-layer isolation. It must not prescribe concrete database or adapter imports inside domain models (`internal/<component>/domain/`), as defined in [code-architecture](.agents/rules/code-architecture.md).
3.  **Cloud-Native & API-First Architecture**: Validate that the spec enforces statelessness, circuit breakers, deadlines, multitenancy isolation, versioned API-First REST exposure, and Contract-Based decoupling per [cloud-first](.agents/rules/cloud-first.md).
4.  **Database & Stack Compliance**: Validate that data models adhere to PostgreSQL via `pgx/v5` and migrations via `goose/v3` (never GORM) per [nonfunctional](.agents/rules/nonfunctional.md).
5.  **Observability & Health Hooks**: Verify that the spec includes dependency checks for readiness probes, standard JSON logs via zerolog, OTel traceparent context propagation, and Prometheus metric hooks per [observability](.agents/rules/observability.md) and [k8s-best-practices](.agents/rules/k8s-best-practices.md).

---

## Step 2: Formulate the Technical Feedback Report
Do not simply rewrite the specification or begin implementation. Compile your findings into a structured, technical report to present to the User:

*   **Strengths**: Identify what elements are well-defined, robust, and compliant.
*   **Identified Gaps & Risks**: List specific database schema / migration needs, missing REST status codes, rate-limiting quotas, OTel propagation gaps, or missing hexagonal port interfaces.
*   **Collaborative Refinement Questions**: Formulate concise, actionable questions to query the User on resolving the identified gaps.

---

## Step 3: Interactive Dialogue & Iterative Refining
Engage in a cooperative dialogue with the User to address gaps:
1.  Discuss each collaborative question with the User.
2.  Based on the User's input, update the GitHub issue description (`gh issue edit <ID> --body "..."`).
3.  Re-run the Compliance Check (Step 1) against the updated issue description.

---

## Step 4: Spec Acceptance Transition
Once you and the User have resolved all gaps and are in full agreement:
1.  Ask the User for approval to accept the specification.
2.  **ACCEPTED Transition**: Upon approval, update the issue status label:
    ```bash
    gh issue edit <ID> --remove-label "status:draft" --add-label "status:accepted"
    ```
3.  The issue is now ready for assignment, branch creation, and TDD execution via `sdd-task-execution.md`.
