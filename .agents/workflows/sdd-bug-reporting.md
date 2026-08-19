---
trigger: manual
description: Reporting, triaging, and executing bug fixes through GitHub Issues and TDD cycles.
---

# SDD Workflow: Bug Reporting & Resolution (TDD Defect Lifecycle)

This workflow defines the step-by-step process for documenting, staging, and executing a bug fix via GitHub Issues and Pull Requests, fully aligned with the [Project Constitution](specs/constitution.md).

In Tacito Square, **a bug is treated just like a task** — it encapsulates both the specification of the defect (expected vs. actual behavior) and the TDD fix requirements.

---

## Step 1: Bug Analysis & Triage
When a defect, crash, or test failure is identified:
1.  **Isolate the Defect**: Inspect logs, test outputs, or database queries to pinpoint the root cause.
2.  **Define the Scope**: Identify which components (`keeper`, `agent`, `operator`, `bff`, `ui`, `shared`, `deploy`) are affected, and whether any architectural rules or specifications are violated.

---

## Step 2: GitHub Bug Issue Creation
Create a GitHub Bug Issue using `.github/ISSUE_TEMPLATE/bug.yml` or the GitHub CLI:

```bash
gh issue create \
  --title "[BUG]: <Short defect title>" \
  --label "type:bug,status:accepted,comp:<component>,severity:<low|medium|high|critical>" \
  --body "## Defect Description
<Detailed explanation of incorrect behavior>

## Reproduction Steps & Logs
1. ...
2. ...

## Expected Behavior
The system MUST ...

## TDD Fix Plan
- [ ] Write reproducing failing test (RED)
- [ ] Implement root cause fix (GREEN)
- [ ] Clean up and verify no regressions (REFACTOR)
"
```

---

## Step 3: Execution via TDD Operational Loop
Follow the standard TDD cycle on a feature branch (Principle P2: "Tests are written FIRST. Implementation follows. No exception"):

1.  **RED Phase**: Write an automated test replicating the defect (unit, integration, or HTTP test). Run the test to witness a clean, expected **test failure (RED)** proving the bug is reproduced.
2.  **GREEN Phase**: Implement the minimum possible code change using approved stack patterns (`pgx/v5`, `goose/v3`, Gin test recorder) to make the test pass (**GREEN**).
3.  **REFACTOR Phase**: Refactor code logic or formatting while keeping test contracts frozen and completely GREEN.

---

## Step 4: PR Submission & Bug Closure
Once all unit and integration tests pass:
1.  Submit a Pull Request referencing `Fixes #<ID>` using `gh pr create`.
2.  Upon review and merge by the user, the PR closes the issue and completes the bug resolution.
