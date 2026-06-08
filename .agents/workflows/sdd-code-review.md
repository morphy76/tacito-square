---
description: 
---

# SDD Workflow: Agent Code Review & Hotspot Refactoring

This workflow defines the step-by-step process for reviewing and refactoring the codebase to improve **maintainability**, **readability**, and **context efficiency** in a focused, incremental manner. 

Aligned with the [Project Constitution](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/constitution.md), this workflow guarantees system stability by preserving all existing tests while focusing agent intelligence on the most complex files ("hotspots").

---

## Governing Constraints

> [!IMPORTANT]
> **Constraint 1: Target Languages Only**
> This workflow only applies to **Go** (`.go`) and **TypeScript** (`.ts`, `.tsx`) files.
>
> **Constraint 2: No Test Modifications**
> Existing test files (`*_test.go`, `*.spec.ts`, etc.) **MUST NOT** be modified. The current test suite acts as an immutable safety net protecting the behavior of the system.
>
> **Constraint 3: Single Hotspot Focus**
> Never refactor multiple hotspots in a single pass. The agent must operate on exactly **one** top hotspot at a time to ensure high context efficiency and reduce risk.
>
> **Constraint 4: Priority Queue Persistence**
> All identified but unworked hotspots must be stored in `specs/hotspots.md` so they are prioritized for the next run.

---

## Step-by-Step Workflow

### Step 1: Initialize or Load the Hotspots Registry
Before performing any analysis, check if the hotspots registry exists:
- Open [specs/hotspots.md](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/hotspots.md).
- If the file does not exist, initialize it using the **Hotspots Registry Template** below.
- If it exists and contains outstanding unworked hotspots, **skip the codebase-wide scan** (Step 2) and proceed to select the top-priority item (Step 3).

### Step 2: Codebase Hotspot Identification Scan
If the registry is empty or a fresh scan is requested:
1. **Locate Long Files**: Search for Go and TypeScript files (excluding `vendor/`, `node_modules/`, `zz_generated*`, and test files) that have **more than 500 lines of code (LOC)**.
2. **Identify Complexity Warnings**: Check for files containing markers of high cognitive/structural complexity:
   - Deep nesting levels (e.g., nested `if`/`for` loops > 3 levels deep).
   - High cognitive complexity (e.g., God functions, excessive conditional paths).
   - Violations of Hexagonal Architecture boundaries (e.g., domain importing adapters).
3. **Prioritize the List**: Sort the identified files in descending order of urgency based on:
   - Size (LOC).
   - Structural/architectural issues.
   - Frequency of modification (if known).
4. **Update the Registry**: Write the sorted list to [specs/hotspots.md](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/hotspots.md). Keep all unworked hotspots in the **Active Hotspots Queue**.

### Step 3: Select the Top Hotspot
1. Pick the very first file from the **Active Hotspots Queue** in `specs/hotspots.md`.
2. Move its status in the registry table from `QUEUED` to `IN_PROGRESS`.
3. Provide a clear justification for why this file is selected and link it.

### Step 4: Decompose and Plan the Refactoring Task
1. Create a task file for the refactoring: `specs/tasks/refactor/TASK-REFACTOR-{FILENAME}.md`.
2. Clearly define the objectives focusing on:
   - **Maintainability**: Split God functions, reduce nesting, extract clean internal helpers, deduplicate logic.
   - **Readability**: Improve naming, verify structured log clarity, add concise godoc comments/docstrings.
   - **Context Efficiency**: Streamline imports, simplify struct/interface signatures, and ensure the file is as compact and readable as possible.
3. **Include the Immutable Test Rule**: The task description must explicitly mandate that **no existing tests will be modified**.
4. Present the task file to the User and **wait for approval** before writing any code.

### Step 5: Execute via the Refactoring TDD Loop
1. **Establish Green Baseline**: Run the existing test suite (`make test` or equivalent) to ensure all tests pass.
2. **Perform Incremental Edits**: Implement refactoring changes in the target file.
3. **Verify Behavior**: Periodically run the test suite to ensure the baseline remains fully **GREEN**. If tests fail, resolve the regression in the implementation code; **never modify the test code**.
4. **Lint and Format**: Run the project linter (`make lint`) to verify syntax, style, and formatting.

### Step 6: Update Queue and Close Task
1. Update the status of the refactored file in [specs/hotspots.md](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/hotspots.md):
   - Set status to `RESOLVED`.
   - Record the reduction in LOC or complexity improvement.
2. Commit the changes and the updated registry.
3. Present the results to the User for task verification.

---

## HOTSPOTS REGISTRY TEMPLATE (`specs/hotspots.md`)

```markdown
# Codebase Hotspots Registry

> Track and prioritize Go and TypeScript files with high complexity or length (>500 LOC) to refactor them incrementally.

## Active Hotspots Queue

| Priority | File Path | Lang | LOC | Complexity Issues | Status |
| :---: | :--- | :---: | :---: | :--- | :---: |
| 1 | [internal/keeper/adapters/outbound/postgres/agent_repository.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/outbound/postgres/agent_repository.go) | Go | 550 | N+1 queries, nested iteration | QUEUED |
| 2 | [internal/agent/adapters/outbound/qdrant/ltm_adapter.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/adapters/outbound/qdrant/ltm_adapter.go) | Go | 420 | Complex filter builder, manual mappings | QUEUED |

## Refactoring History

| Resolved Date | File Path | Lang | Original LOC | New LOC | Improvements Achieved | Task Link |
| :--- | :--- | :---: | :---: | :---: | :--- | :---: |
| 2026-06-07 | [internal/keeper/application/service/agent_service.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/application/service/agent_service.go) | Go | 610 | 480 | Split CRD operations into helper domain service | [TASK-REFACTOR-agent_service](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/tasks/refactor/TASK-REFACTOR-agent_service.md) |
```

---

## REFACTOR TASK TEMPLATE (`specs/tasks/refactor/TASK-REFACTOR-{name}.md`)

```markdown
# TASK-REFACTOR-{name}: Refactor {Filename}

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-REFACTOR-{name}                        |
| Status        | DRAFT                                       |
| Target File   | [{Filename}](file:///absolute/path/to/file)  |
| Baseline Tests| All existing tests MUST pass without changes |

## Description
Detailed plan for clean-up, structure simplification, and size reduction of the target file.

## Work Items
1. **Baseline Phase**:
   - Verify all existing tests pass.
2. **Refactor Phase**:
   - [ ] Split {function_x} to reduce cognitive complexity.
   - [ ] Extract internal struct/helper for {concept_y}.
3. **Verification Phase**:
   - Run existing tests to ensure they are 100% green.
   - Run `make lint` to confirm codebase styling compliance.

## Acceptance Criteria
1. No existing unit/integration/contract test is modified.
2. The target file is cleaner, has fewer lines of code, and shows improved readability.
3. Lint checks pass cleanly.
```
