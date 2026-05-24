# SDD Workflow: Verification & Resolution Loop

This workflow defines the mandatory verification process that AI development agents MUST execute upon completing any implementation task. It enforces strict compliance with specifications and rules before concluding work, fully aligned with the [Project Constitution](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/constitution.md) and tracked via the master [Specs Index](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/INDEX.md).

---

## Step 1: Multidimensional Verification
Upon finishing code changes, the agent MUST evaluate the implementation across three primary axes to guarantee absolute compliance:

### Axis A: Fidelity to Functional Specification
*   Open the corresponding functional specification under `specs/functional/`.
*   Systematically verify each numbered requirement in the **Specification** section.
*   Cross-check each criterion in the **Acceptance Criteria** section.
*   Run every test scenario outlined in the **Test Plan**.

### Axis B: Codebase Structural & Quality Integration
*   Run the test suite using `make test` to verify all tests (unit, integration) compile and pass successfully.
*   Check that compiler flags are correctly set (`CGO_ENABLED=0`, `-ldflags="-s -w"`) and the code builds cleanly with zero lint warnings.
*   Verify GORM/Postgres migrations are registered correctly, naming conventions are respected, and no temporary directories have been written outside the workspace.

### Axis C: Active Agent Rules Compliance
Review your code changes against all active rules under `.agents/rules/`:
1.  **[spec_driven_development](file:///Users/R.Pasquini/Projects/side/tacito-square/.agents/rules/spec_driven_development.md)**: Confirm that no code was written without an approved spec/task and that TDD was strictly executed in accordance with Principles P1 & P2 of the [Project Constitution](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/constitution.md).
2.  **[cloud_first](file:///Users/R.Pasquini/Projects/side/tacito-square/.agents/rules/cloud_first.md)**: Ensure circuit breakers, timeouts/deadlines, retries, statelessness, multitenancy JWT/headers, and API-First REST exposure are implemented for any new network logic.
3.  **[k8s_best_practices](file:///Users/R.Pasquini/Projects/side/tacito-square/.agents/rules/k8s_best_practices.md)**: Ensure HPA templates are added, parallel `/healthz` and `/readyz` dependency checks are implemented, and Distroless Docker bases are used.
4.  **[observability](file:///Users/R.Pasquini/Projects/side/tacito-square/.agents/rules/observability.md)**: Verify Prometheus `/metrics` exposition, OpenTelemetry context propagation, and structured JSON logs using zerolog/winston with correlated `trace_id`/`span_id` contexts.
5.  **[code_architecture](file:///Users/R.Pasquini/Projects/side/tacito-square/.agents/rules/code_architecture.md)**: Ensure Hexagonal DDD layering is preserved (no adapter imports in domain/application) and Go concurrency primitives are managed with proper contexts.
6.  **[nonfunctional](file:///Users/R.Pasquini/Projects/side/tacito-square/.agents/rules/nonfunctional.md)**: Check stack locks, Makefile phony targets, and Gin framework standards.

---

## Step 2: Interactive Gap Analysis & Resolution
If you discover **any gap, lack, or rule violation** during Step 1, you MUST compile them into a structured report and **prompt the User interactively for feedback**. 

### Mandated Prompt Structure
Present the list of discovered gaps or lacks to the User, and ask the User to select how to resolve them:

> ### Discovered Gaps & Compliance Gaps
> 
> *   **Gap 1**: {Describe lack or violation} (e.g. Missing `trace_id` correlation on database log entries)
> *   **Gap 2**: {Describe lack or violation} (e.g. Dockerfile does not use Google Distroless runtime base)
> 
> Please select how you would like me to resolve these lacks:
> 1.  **(Option A) Immediate Resolution**: Fix these lacks immediately in this task session using the TDD cycle.
> 2.  **(Option B) Formal BUG Specification**: Create a formal, self-contained BUG task file (`BUG-M<Milestone>.<Number>.md`) to track these lacks for future iteration, and register them in the index.

### Execution of Resolution Options
*   **If User selects Option A**: Return to the **Task Execution Workflow** in `IN_PROGRESS` status, implement the required tests and code fixes under TDD, and run the Verification workflow again.
*   **If User selects Option B**: Create a BUG task document following the **Bug Reporting Workflow** (`sdd_bug_reporting.md`), register it in [specs/INDEX.md](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/INDEX.md) as `OPEN`, and proceed to present the current main task as `IMPLEMENTED` for review.

---

## Step 3: Specs Index Status Transition (VERIFIED) & Milestone Transitions
Finally, verify the overall milestone lifecycle state. Under Section 5 (**Milestone Lifecycle**) of the [Project Constitution](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/constitution.md), milestones automatically transition based on functional specs states:

1.  **Specs Index Update (VERIFIED)**: Once the User has reviewed the completed work and given their approval, open the master [Specs Index](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/INDEX.md) and update the status of the **parent functional specification (FR)** to `VERIFIED` in its milestone specifications table.
2.  **✔️ IMPLEMENTED Milestone Verification**: Verify if every required functional spec in the current milestone has achieved at least `IMPLEMENTED` status.
3.  **🎉 COMPLETED Milestone Verification**: Once every functional spec in the milestone achieves `VERIFIED` status in the index, you MUST update the milestone's overall status to `COMPLETED` in the master [Specs Index](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/INDEX.md).
