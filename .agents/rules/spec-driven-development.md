---
trigger: always_on
globs: ["*"]
description: Spec-Driven Development (SDD) compliance and Test-Driven Development (TDD) operational practices with GitHub Issues and PRs.
---

# Spec-Driven & Test-Driven Development (SDD/TDD) Guidelines

This rule enforces strict adherence to the project's development workflow, combining Spec-Driven Development (SDD) compliance with Test-Driven Development (TDD) operational execution as specified in the [Project Constitution](specs/constitution.md).

## 1. Spec-Driven Development Compliance (GitHub Issues)

Every code change must trace back to a defined feature specification issue and structured task.

- **No Code Without a Spec:** Prohibit implementation code changes without a corresponding accepted GitHub Issue (`type:spec`).
- **Review and Approval Loops:**
  - An issue in `status:draft` must be reviewed collaboratively with the user.
  - No issue can be started (`status:in-progress`) without explicit review and approval from the spec owner (the User), moving it to `status:accepted`.
  - The standard issue lifecycle is:
    ```
    status:draft -> status:accepted -> status:in-progress -> status:implemented (PR open) -> status:verified / closed (PR merged)
    ```
- **Spec Atomicity:** Every specification issue MUST define exactly one atomic, testable capability.
- **Spec Immutability (Principle P6):** Once a specification issue is verified and closed on `main`, it is frozen. Behavior changes require a new superseding issue referencing `Supersedes #<issue-id>`.

## 2. Standard GitHub Development Lifecycle

When working on a feature, bug, or task, strictly follow this collaborative workflow:

1. **Review & Assignment:**
   * Review the issue details collaboratively with the user (`gh issue view <ID>`).
   * Assign the issue to the human owner (`gh issue edit <ID> --add-assignee morphy76`).
   * Switch the status label to `status:accepted` upon approval.
2. **Branch Creation & Engagement:**
   * Check out the designated feature branch tracking the issue (e.g., `git checkout <branch-name>`).
   * Transition status to `status:in-progress`.
3. **TDD Operational Execution (Red → Green → Refactor):**
   * Follow the strict Red-Green-Refactor loop on the feature branch.
4. **Pull Request Submission:**
   * Push changes to origin.
   * Open a Pull Request referencing `Fixes #<ID>` using `gh pr create`.
   * Transition label to `status:implemented`.
5. **Review & Merge:**
   * The human user reviews and merges the PR.
   * Merging closes the issue automatically and deletes the feature branch.

## 3. TDD Operational Execution (Red → Green → Refactor)

When executing any task or issue on a branch:

1. **RED Phase (Write Tests First):**
   * Write unit, integration (with testcontainers), or contract tests *first* before modifying any implementation code.
   * Tests must clearly define the interface contracts, dependencies, and requirements.
   * Run `go test ./...` and verify that the new tests fail (RED) as expected.
2. **GREEN Phase (Implement Minimum Code):**
   * Write the minimum amount of code required to satisfy the newly added tests.
   * Do not over-engineer or add out-of-scope functionality.
   * Verify that the test suite compiles and runs successfully (GREEN).
3. **REFACTOR Phase (Clean & Generalize):**
   * Clean up the code, optimize performance, and generalize abstractions.
   * Do not modify the test assertions during this phase.
   * Verify that the test suite remains completely GREEN.

## 4. Task & Work Item Decomposition

- **Logical Boundary Organization:** Decompose large specifications into sub-issues or tasks (`type:task`) structured strictly around logical system boundaries (packages, DDD aggregates, modules, components).
- **No Organization by TDD Phase:** Tasks MUST NOT be split into separate issues for "write test" vs. "write implementation". TDD is purely an operational developer methodology used to execute a task, not an organizational boundary.

---

## Developer Checklists & Verifications

- [ ] Is there an accepted GitHub Issue (`type:spec`, `status:accepted`) for this feature?
- [ ] Am I working on the dedicated feature branch linked to the issue?
- [ ] Did I write unit/contract tests *first* and see them fail (RED) before implementing logic?
- [ ] Does my implementation respect Hexagonal architecture and domain boundaries?
- [ ] Does my PR description reference `Fixes #<ID>`?
