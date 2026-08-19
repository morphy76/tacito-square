---
trigger: manual
description: Executing tasks and specs through the TDD lifecycle on Git feature branches and Pull Requests.
---

# SDD Workflow: Task Execution (TDD Lifecycle & Pull Request Workflow)

This workflow defines the step-by-step process for executing a specification or task using **Test-Driven Development (TDD)** on dedicated Git feature branches and delivering changes via Pull Requests, fully aligned with the [Project Constitution](specs/constitution.md).

---

## Step 1: Pre-Execution Alignment & Branch Setup
Before writing any code:

1.  **Verify Status**: Ensure the target issue has label `status:accepted` (or is an assigned bug/task) and is assigned to the user (`morphy76`).
2.  **Branch Setup**: Ensure the remote branch is created and checked out locally:
    ```bash
    git checkout <branch-name>
    ```
3.  **Update Label**: Mark the issue as in progress:
    ```bash
    gh issue edit <ID> --remove-label "status:accepted" --add-label "status:in-progress"
    ```
4.  **Decompose by Logical Boundary**: If the issue is large, decompose it logically by subsystem or DDD package (never by TDD phase).

---

## Step 2: Test-Driven Development (TDD) Operational Loop
Execute the core development cycle on the branch (Principle P2: "Tests are written FIRST. Implementation follows. No exception"):

### A. RED Phase (Write Tests First)
1.  Write unit, contract, or integration tests **before** modifying any implementation logic.
2.  Adhere to approved database and architecture standards: PostgreSQL with `pgx/v5`, migrations with `goose/v3` (never GORM), pure domain layer with zero adapter imports.
3.  Run the tests (`go test ./...`) and verify they compile and **FAIL (RED)** as expected.

### B. GREEN Phase (Implement Minimum Code)
1.  Write the absolute **minimum amount of code** required to make the failing tests pass (GREEN).
2.  Avoid speculative code or out-of-scope refactoring.
3.  Verify that the entire test suite passes successfully.

### C. REFACTOR Phase (Clean & Generalize)
1.  Clean up code duplication, improve naming, optimize performance, and ensure strict Hexagonal layer isolation.
2.  **CRITICAL CONSTRAINT**: Do not modify test assertions during refactoring; tests remain a frozen contract.
3.  Verify the test suite remains completely GREEN.

---

## Step 3: Integration & Infrastructure Verification
If external or infrastructural dependencies are involved (PostgreSQL, Redis, Qdrant, NATS):
1.  Run integration tests tagged with `integration` using `testcontainers-go`:
    ```bash
    go test ./... -tags=integration -v
    ```
2.  Run linter and quality checks:
    ```bash
    make lint
    make test
    ```

---

## Step 4: Pull Request Submission
Once all tests and quality checks pass:

1.  **Commit & Push**:
    ```bash
    git add .
    git commit -m "feat(<component>): <concise description> (Fixes #<ID>)"
    git push origin <branch-name>
    ```
2.  **Create Pull Request**:
    ```bash
    gh pr create \
      --title "<Component>: <Title>" \
      --body "## Description
<Summary of changes>

Fixes #<ID>

## Checklist
- [x] Unit/Integration tests added (TDD RED -> GREEN -> REFACTOR)
- [x] Hexagonal architecture boundaries verified
- [x] Local test suite passes (make test)"
    ```
3.  **Update Issue Label**:
    ```bash
    gh issue edit <ID> --remove-label "status:in-progress" --add-label "status:implemented"
    ```
4.  Present the PR link to the user for review and merge. Merging the PR closes the issue and completes the lifecycle.