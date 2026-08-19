---
trigger: manual
description: Reviewing and refactoring codebase hotspots, maintaining clean hexagonal architecture and zero test regressions.
---

# SDD Workflow: Agent Code Review & Hotspot Refactoring

This workflow defines the step-by-step process for reviewing and refactoring the codebase to improve **maintainability**, **readability**, and **context efficiency** in a focused, incremental manner via GitHub Pull Requests.

Aligned with the [Project Constitution](specs/constitution.md), this workflow guarantees system stability by preserving all existing tests while focusing agent intelligence on the most complex files ("hotspots").

---

## Governing Constraints

> [!IMPORTANT]
> **Constraint 1: Target Languages Only**
> This workflow applies to **Go** (`.go`) and **TypeScript** (`.ts`, `.tsx`) files.
>
> **Constraint 2: No Test Modifications**
> Existing test files (`*_test.go`, `*.spec.ts`, etc.) **MUST NOT** be modified. The current test suite acts as an immutable safety net protecting the behavior of the system.
>
> **Constraint 3: Single Hotspot Focus**
> Never refactor multiple hotspots in a single pass. The agent must operate on exactly **one** top hotspot at a time to ensure high context efficiency and reduce risk.

---

## Step-by-Step Workflow

### Step 1: Hotspot Identification & Selection
1. Locate files with >500 LOC or high cognitive complexity (deep nesting >3 levels, God functions).
2. Create or select a refactoring task issue (`type:refactor` or `type:task`).

### Step 2: Establish Baseline
1. Run existing test suite (`make test`) and ensure all tests pass (GREEN).
2. Verify linter status (`make lint`).

### Step 3: Incremental Refactoring
1. Refactor logic to simplify functions, improve variable naming, and ensure Hexagonal boundary decoupling.
2. Re-run tests frequently to ensure the baseline remains 100% GREEN.

### Step 4: PR Creation
1. Open a Pull Request referencing the refactoring issue.
2. Confirm that tests and linter pass on CI.
