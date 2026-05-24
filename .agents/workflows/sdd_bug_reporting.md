# SDD Workflow: Bug Reporting (Self-Contained Task)

This workflow defines the step-by-step process for documenting, staging, and executing a bug fix, fully aligned with the [Project Constitution](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/constitution.md) and tracked via the master [Specs Index](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/INDEX.md).

In Tacito Square, **a bug is treated just like a task** — a single self-contained document that encapsulates both the specification of the defect (expected vs actual behavior) and the task requirements. (Section 3 of the [Project Constitution](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/constitution.md)).

---

## Step 1: Bug Analysis & Triage
When a defect, crash, or test failure is identified:
1.  **Isolate the Defect**: Run logs, database inspections, or GORM schema queries to pinpoint the root cause.
2.  **Define the Scope**: Identify which components (`keeper`, `agent`, `operator`, `bff`) and files are affected, and which principles/specifications of the [Project Constitution](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/constitution.md) are violated.

---

## Step 2: Bug Document Creation & Indexing
Create a self-contained BUG document following the exact template below:

1.  **Path**: `specs/tasks/M<Milestone-Number>.BUG<Bug-Number>/BUG-M<Milestone-Number>.<Bug-Number>.md`
2.  **Initial Status**: Set `Status` to `OPEN`.
3.  **Specs Index Registration**: Open the master [Specs Index](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/INDEX.md), find the corresponding Milestone's Bugs table, and append a new entry for the bug mapping its ID, Title, status `OPEN`, severity, and clickable file link.

---

## Step 3: Execution via TDD Operational Loop
Under the "bug is just like a task" philosophy, execution follows the exact same Test-Driven Development (TDD) cycle as task execution. This directly implements Principle P2 (**TDD**) of the [Project Constitution](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/constitution.md):
1.  **RED Phase**: Write tests replicating the defect. Run the tests to witness a clean, expected **test failure (RED)** showing that the bug is reproduced.
2.  **GREEN Phase**: Implement the minimum possible code change to resolve the defect and make all tests pass successfully (**GREEN**).
3.  **REFACTOR Phase**: Refactor code logic, efficiency, or formatting while keeping test contracts frozen and completely GREEN.

---

## Step 4: Present for Bug Closure
Once resolved:
1.  **Accepted Transition**: Move the BUG document status to `CLOSED` in its own metadata header.
2.  **Specs Index Update**: Open the master [Specs Index](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/INDEX.md) and update the status of the bug to `CLOSED` inside the corresponding Milestone's Bugs table.
3.  Present the solution and clickable file link to the User, and wait for confirmation (adhering to Section 3 of the [Project Constitution](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/constitution.md)).

---

## BUG DOCUMENT TEMPLATE

```markdown
# BUG-M{Milestone}.{Bug-Number}: {Title}

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M{Milestone}.{Bug-Number}                                      |
| Status        | OPEN                                                               |
| Severity      | {LOW|MEDIUM|HIGH}                                                   |
| Milestone     | M{Milestone} — {Milestone Name}                                    |
| Affects       | {file/component path}                                              |
| Violates      | SPEC-FR-M{X}.{Y}, SPEC-NFR-{Z}                                     |
| Discovered    | {Context, logs, or readyz readiness test failures}                  |

## Problem Statement

{Detailed explanation of the current incorrect behavior, root cause analysis, GORM/Postgres config conflicts, or TLS mounting issues.}

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| {Component Name} | {Path to File} | {Specific code lack, GORM Gaps, or configuration error} |

## Impact

1. {Logical impact on security, database state, or connectivity.}
2. {Impact on downstream component readiness probes (/readyz).}

## Expected Behaviour

1. {Requirement 1: The system MUST...}
2. {Requirement 2: GORM migration scripts MUST...}

## Acceptance Criteria

1. {Concrete verification check, e.g. running connection verification script connects successfully.}
2. {Readiness checks pass successfully and /readyz returns 200 OK.}
```
