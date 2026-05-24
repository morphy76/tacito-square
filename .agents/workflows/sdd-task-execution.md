# SDD Workflow: Task Execution (TDD Lifecycle)

This workflow defines the step-by-step process for planning, decomposing, executing, and refactoring a system change using **Test-Driven Development (TDD)** within defined logical boundaries, fully aligned with the [Project Constitution](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/constitution.md) and tracked via the master [Specs Index](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/INDEX.md).

---

## Step 1: Task Planning & Boundary Organization
Before writing any code, decompose your work into structured task files. In accordance with Section 3 (**Rules**) of the [Project Constitution](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/constitution.md):

1.  **Decompose by Logical Boundary**: Organize tasks strictly around package boundaries, DDD aggregates, module boundaries, components, or functional subsystems. (Constitution Section 3, Rule 7).
2.  **Strict Avoidance**: **NEVER** organize tasks by TDD phases (e.g., do not make separate tasks for "write tests" vs. "write implementation"). TDD is purely an operational developer methodology used to execute a task, not an organizational boundary. (Constitution Section 3, Rule 8).
3.  **Task File Generation**: Create a task spec file under the corresponding milestone/feature folder:
    -   **Path**: `specs/tasks/M<Milestone-Number>.<Feature-Number>/TASK-M<Milestone-Number>.<Feature-Number>.<Task-Number>.md`
    -   **Initial Status**: Set status to `DRAFT`.
4.  **Interactive Approval Loop**: Present the task plan to the User with a direct clickable link. **STOP and wait** for the User's explicit review and approval before moving the status to `IN_PROGRESS` and modifying any codebase file. (Constitution Section 3, Rule 3).
5.  **Index Status Transition (IN_PROGRESS)**: Once the User approves the task, set the task file status to `IN_PROGRESS`. Simultaneously, you MUST open the master [Specs Index](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/INDEX.md) and update the status of the **parent functional specification (FR)** to `IN_PROGRESS` inside the milestone specification table.

---

## Step 2: Test-Driven Development (TDD) Operational Loop
Once the task is approved and set to `IN_PROGRESS`, you must strictly follow the operational TDD cycle. This directly implements Principle P2 (**TDD**) of the [Project Constitution](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/constitution.md) ("Tests are written FIRST. Implementation follows. No exception"):

### A. RED Phase (Write Tests First)
1.  Write unit, integration, or contract tests **before** creating or modifying any business logic or implementation code.
2.  Verify that your new tests compile successfully but **fail (RED)** when executed against the existing codebase.
3.  Do not proceed to the Green phase until you have witnessed a clean, expected test failure.

### B. GREEN Phase (Implement Minimum Code)
1.  Write the absolute **minimum amount of code** required to make the failing tests compile and pass successfully (GREEN).
2.  Avoid over-engineering, writing speculative code, or implementing capabilities outside the strict scope of the current task.
3.  Verify that the entire test suite passes successfully.

### C. REFACTOR Phase (Clean & Generalize)
1.  Review the newly added code. Clean up code duplication, improve naming, optimize performance, and generalize abstractions.
2.  **CRITICAL CONSTRAINT**: Do not modify any test assertions or test logic during this phase. The tests must remain a frozen contract.
3.  Verify that the test suite remains completely GREEN.

---

## Step 3: Present for Task Completion Approval
Once the task is complete:
1.  Update the task file status to `IMPLEMENTED` in accordance with the transition states diagram in Section 3 of the [Project Constitution](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/constitution.md).
2.  **Index Status Transition (IMPLEMENTED)**: Open the master [Specs Index](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/INDEX.md) and update the status of the **parent functional specification (FR)** to `IMPLEMENTED` inside the milestone specification table.
3.  Refer the User to the task file link.
4.  **STOP and wait** for the User's verification and approval to mark the task `VERIFIED`. (Constitution Section 3, Rule 4).

---

## TASK FILE TEMPLATE

```markdown
# TASK-M{Milestone}.{Feature}.{Task}: {Title}

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M{Milestone}.{Feature}.{Task}          |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M{Milestone}.{Feature}              |
| Depends On    | TASK-M{X}.{Y}.{Z}, ... (or none)             |

## Description

{Detailed description of the task scope, GORM mappings, HTTP controllers, or interface boundaries covered.}

## Work Items

1. **RED Phase**:
   - {List of tests to create first, e.g. create internal/keeper/domain/llm_binding_test.go verifying input validations.}
   - {List of database/integration tests to create, e.g. verify repository connection pools.}
2. **GREEN Phase**:
   - {List of minimal implementation steps, e.g. implement llm_binding.go validation rules.}
   - {Define repository interface and write Postgres GORM implementation.}
3. **REFACTOR Phase**:
   - {Refactor plan, e.g. optimize query allocations, improve error wraps.}

## Acceptance Criteria

1. {Concrete test outputs and success benchmarks, e.g. all GORM unit and integration tests pass successfully.}
2. {Hexagonal architecture boundaries are fully verified with zero import violations.}
```
