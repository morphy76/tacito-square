---
trigger: manual
description: SDD workflow to verify the functional implementation of a milestone or PR by running E2E and component-level verification and generating a test report.
---

# SDD Workflow: Milestone Verification (Tester Lifecycle)

This workflow defines the step-by-step process for verifying the functional implementation of a milestone or Pull Request. Aligned with the [Project Constitution](specs/constitution.md), the verification process validates that all functional specification issues meet their Acceptance Criteria and Test Plans in a live, integrated environment.

---

## Governing Constraints

> [!IMPORTANT]
> **Constraint 1: Local Cluster Context & Tooling**
> Verification operations run against the local Rancher Desktop cluster. Ensure your `kubectl` context is set to `rancher-desktop` and operations target the `tacito` namespace. The agent is authorized to use `kubectl`, `curl`, and NATS CLI clients/utilities to interact with deployed resources.
>
> **Constraint 2: No Source Code Mutations**
> The tester agent MUST NOT write or modify application implementation code. If code changes are required, document the defect via `sdd-bug-reporting.md`.
>
> **Constraint 3: Token & Context Efficiency**
> Use `gh issue view <ID>` to extract only the Acceptance Criteria and Test Plan.

---

## Step-by-Step Workflow

### Step 1: Milestone & Spec Discovery
1. **Identify the Milestone or Issue**: Retrieve the target milestone (e.g., `Consolidate`) or PR from user input.
2. **Scan Milestone Requirements**:
   ```bash
   gh issue list --milestone "<Milestone>" --state all
   ```
3. **Extract Target Sections**: For each specification, read the `Acceptance Criteria` and `Test Plan`.

### Step 2: Environment Readiness
1. Verify target pods (`keeper`, `operator`, `bff`) are running in the `tacito` namespace:
   ```bash
   kubectl get pods -n tacito
   ```
2. If pods are missing or need rebuild:
   ```bash
   make build
   make docker-build
   make helm-install
   ```

### Step 3: Run Automated & Integration Test Suites
1. Run automated test suites:
   ```bash
   make test
   make test-integration
   make test-contract
   ```
2. For live validation:
   - **REST APIs**: Query `/readyz` or business endpoints with `curl`.
   - **NATS Message Interaction**: Publish and subscribe to test subjects.
   - **Structured Logs Inspection**: Stream and inspect active container logs (`kubectl logs -n tacito -l app=<component> --tail=50`).

### Step 4: Verification Report & Status Promotion
1. Summarize findings in a concise report.
2. If all acceptance criteria pass:
   - Update issue status labels to `status:verified`:
     ```bash
     gh issue edit <ID> --remove-label "status:implemented" --add-label "status:verified"
     ```