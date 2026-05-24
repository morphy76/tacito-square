---
trigger: always_on
glob: **/*.{go,ts,md}
description: Spec-Driven Development (SDD) compliance and Test-Driven Development (TDD) operational practices.
---

# Spec-Driven & Test-Driven Development (SDD/TDD) Guidelines

This rule enforces strict adherence to the project's development workflow, combining Spec-Driven Development (SDD) compliance with Test-Driven Development (TDD) operational execution as specified in the Project Constitution.

## 1. Spec-Driven Development Compliance

Every code change must trace back to a defined feature specification and structured task.

- **No Code Without a Spec:** Prohibit implementation code changes without a corresponding accepted functional specification file under `specs/functional/`.
  - *Note: Non-functional specs (`specs/nonfunctional/`) are superseded by the active rules under `.agents/rules/`.*
- **No Code Without a Task:** Prohibit starting code implementation without a corresponding task file under `specs/tasks/<FR-ID>/`.
- **Review and Approval Loops:**
  - No task can be started (`IN_PROGRESS`) without explicit review and approval from the spec owner (the User).
  - No task can be marked complete (`IMPLEMENTED` / `VERIFIED`) without explicit review and approval from the spec owner (the User).
- **Spec Atomicity:** Every spec MUST define exactly one atomic, testable capability.
- **Spec Immutability:** Once a spec is verified, it is frozen. Behavior changes require a new superseding spec with complete tracing headers (`Depends On:` and `Supersedes:`).

## 2. Task Organization & Structure

- **Logical Boundary Organization:** Tasks must be structured strictly around logical system boundaries, packages, modules, components, or functional subsystems.
- **No Organization by TDD Phase:** Tasks MUST NOT be organized by TDD phases (e.g., creating separate tasks for "write test" vs. "write implementation"). TDD is purely an operational developer methodology used to execute a task, not an organizational boundary.

## 3. TDD Operational Execution (Red → Green → Refactor)

When executing any task, always use a strict test-driven development approach:

1. **RED Phase (Write Tests First):**
   * Write unit, integration, or contract tests *first* before modifying any implementation code.
   * Tests must clearly define the interface contracts, dependencies, and requirements.
   * Run the test suite and verify that the new tests fail (RED) as expected.
2. **GREEN Phase (Implement Minimum Code):**
   * Write the minimum amount of code required to satisfy the newly added tests.
   * Do not over-engineer or add out-of-scope functionality.
   * Verify that the test suite compiles and runs successfully (GREEN).
3. **REFACTOR Phase (Clean & Generalize):**
   * Clean up the code, optimize performance, and generalize abstractions.
   * Do not modify the test assertions during this phase.
   * Verify that the test suite remains completely GREEN.

## 4. Spec Document Format

Any new or updated specification file MUST follow this metadata structure:

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
...
## Specification
...
## Acceptance Criteria
...
## Test Plan
...
## API Contract (if applicable)
...
## Files Affected
...
```

---

## Developer Checklists & Verifications

- [ ] Is there an approved spec in `specs/functional/` for this feature?
- [ ] Is there an approved task in `specs/tasks/<FR-ID>/` for this change?
- [ ] Has the spec owner explicitly approved starting this task?
- [ ] Are my tasks structured by package/module boundaries rather than TDD phases?
- [ ] Did I write unit/contract tests *first* and see them fail before implementing any logic?
