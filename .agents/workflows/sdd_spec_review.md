# SDD Workflow: Spec Review & Refinement (Cooperative Dialogue)

This workflow defines the mandatory process for reviewing, technical refinement, and cooperative discussion of any drafted functional specification inside the Tacito Square repository, fully aligned with the [Project Constitution](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/constitution.md) and tracked via the [Specs Index](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/INDEX.md).

---

## Step 1: Pre-Review Compliance Check
When a specification is in `DRAFT` status, or when the User requests a spec review, the agent MUST evaluate the draft spec against the project guidelines:

1.  **Fidelity to Format**: Confirm that the specification contains all standard sections (Context, Specification, Acceptance Criteria, Test Plan, API Contract, Files Affected) and exact metadata headers defined in Section 4 of the [Project Constitution](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/constitution.md).
2.  **Hexagonal DDD Decoupling**: Check that the specification strictly maintains domain-layer isolation. It must not prescribe adapter imports inside the domain package (`internal/<component>/domain/`), as defined in [code_architecture](file:///Users/R.Pasquini/Projects/side/tacito-square/.agents/rules/code_architecture.md).
3.  **Cloud-Native & API-First Architecture**: Validate that the spec enforces statelessness, circuit breakers, deadlines, multitenancy isolation, versioned API-First REST exposure, and Contract-Based decoupling per [cloud_first](file:///Users/R.Pasquini/Projects/side/tacito-square/.agents/rules/cloud_first.md).
4.  **Observability & Health Hooks**: Verify that the spec includes liveness/readiness dependency checks, standard JSON logs, OTel traceparent context propagation, and Prometheus metric hooks per [observability](file:///Users/R.Pasquini/Projects/side/tacito-square/.agents/rules/observability.md) and [k8s_best_practices](file:///Users/R.Pasquini/Projects/side/tacito-square/.agents/rules/k8s_best_practices.md).

---

## Step 2: Formulate the Technical Feedback Report
Do not simply rewrite the specification or begin task execution. Compile your findings into a structured, highly technical, and friendly report to present to the User. The report MUST include:

*   **Draft Strengths**: Identify what elements are well-defined, robust, and compliant.
*   **Identified Gaps & Risks**: List specific GORM database schema index lacks, non-optimal REST status codes, missing rate-limiting quotas, OTel propagation gaps, or missing hexagonal adapters interfaces.
*   **Collaborative Refinement Questions**: Formulate 3-5 highly specific, actionable questions to query the User on resolving the identified gaps.

---

## Step 3: Interactive Dialogue & Iterative Refining
Engage in a cooperative dialogue with the User to address the gaps:
1.  Discuss each of your collaborative questions with the User.
2.  Based on the User's input, **directly edit and iterate on the draft specification file** in the workspace.
3.  Re-run the Compliance Check (Step 1) against the edited draft to ensure all gaps have been cleanly resolved.

---

## Step 4: Spec Acceptance Transition
Once you and the User have resolved all gaps and are in full agreement:
1.  Ask the User for final approval to accept the specification.
2.  **ACCEPTED Transition**: Upon approval, update the specification status to `ACCEPTED` in its own metadata header.
3.  **Specs Index Update**: Open the master [Specs Index](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/INDEX.md) and update the status of the specification to `ACCEPTED` inside its milestone table.
4.  Proceed to plan logical task boundaries according to the **Task Execution Workflow** (`sdd_task_execution.md`).
