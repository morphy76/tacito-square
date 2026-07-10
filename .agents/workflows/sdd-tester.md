---
description: SDD workflow to verify the functional implementation of a milestone by running E2E and component-level verification, setting up the Kubernetes/Rancher environment, and generating a test report.
---

# SDD Workflow: Milestone Verification (Tester Lifecycle)

This workflow defines the step-by-step process for verifying the functional implementation of a milestone. Aligned with the [Project Constitution](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/constitution.md), the verification process validates that all functional specifications (FRs) associated with a milestone meet their Acceptance Criteria and Test Plans in a live, integrated environment.

This workflow is optimized to minimize LLM token consumption, run with high execution efficiency, and build a reusable test knowledge base.

---

## Governing Constraints

> [!IMPORTANT]
> **Constraint 1: Local Cluster Context & Tooling**
> Verification operations MUST run against the local Rancher Desktop cluster. Ensure your `kubectl` context is set to `rancher-desktop` and operations target the `tacito` namespace. The agent is authorized to use `kubectl`, `curl`, and NATS CLI clients/utilities to interact with and inspect deployed resources.
>
> **Constraint 2: No Source Code Mutations**
> The tester agent MUST NOT write or modify application implementation code. If code changes are required to fix a failure, the agent should document the issue as a bug report and stop.
>
> **Constraint 3: Comprehensive Verification**
> Every functional specification (FR) listed under the target milestone in the [Specs Index](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/INDEX.md) must be individually evaluated and verified.
>
> **Constraint 4: Token & Context Efficiency**
> To prevent context window bloat, the agent MUST NOT read entire specification files. Use targeted reads (`StartLine` and `EndLine` parameters or grep) to extract only the **Acceptance Criteria** and **Test Plan** sections.
>
> **Constraint 5: Redundancy Prevention**
> Avoid redundant rebuilds and redeployments. If the cluster is already running the latest built version of the codebase, skip container image builds and Helm re-installations.
>
> **Constraint 6: Read-Only Codebase & Git Exploration**
> The agent is encouraged to perform read-only exploration of the codebase (e.g., viewing test suites, mock files, and configuration paths) and Git history (e.g., `git log` or `git diff`) to understand the layout and history of existing tests.
>
> **Constraint 7: Subagent Delegation**
> The agent is encouraged to spawn subagents to execute discrete sub-tasks (e.g., compiling code, parsing specifications, conducting parallel test runs, running specific `curl`/NATS tasks, or monitoring logs). Subagents keep the main agent's context clean and prevent token bloat by returning only high-level summary reports.

---

## Step-by-Step Workflow

### Step 1: Milestone & Spec Discovery (Token-Optimized)
1. **Identify the Milestone**: Retrieve the target milestone (e.g., `M6.5`) from user input.
2. **Scan Milestone Requirements**: View the milestone section in the master [Specs Index](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/INDEX.md). Identify all functional specifications (`SPEC-FR-*`) and unresolved bug tasks.
3. **Extract Target Sections**: For each specification, read ONLY the `Acceptance Criteria` and `Test Plan` sections. Do not read the Context or Specification details unless clarification is needed on a test failure. *Note: You can delegate parsing of different specs to concurrent subagents to speed up discovery.*

### Step 2: Smart Environment Setup (Execution-Optimized)
1. **Check Existing Health**: Verify if the target pods (`keeper`, `operator`, `bff`) are already running in the `tacito` namespace. Query their `/readyz` endpoints.
2. **Conditional Rebuild & Deploy**: 
   - If pods are healthy and run the current commit state, proceed directly to Step 3.
   - If out of sync or unhealthy, execute the build pipeline:
     ```bash
     make build
     make docker-build
     make helm-install
     ```
     *(Note: This build and deployment process can be delegated to a background subagent).*
3. **Wait for Readiness**: Block until all pods show `Ready` status via `kubectl get pods -n tacito`.

### Step 3: Test Layout Discovery & Exploration (Knowledge-Consolidated)
1. **Consult Existing Knowledge**: Check the test knowledge base under `specs/reports/test-knowledge-base/` to see if the testing patterns, mock usages, or command sequences for the components under test have already been documented.
2. **Read-Only Exploration**: If knowledge is missing or incomplete, perform codebase/Git exploration. *You may spawn subagents to analyze different directories or components in parallel.*
3. **Consolidate Test Knowledge**: Document findings in a markdown file at `specs/reports/test-knowledge-base/<component>-testing-notes.md`. Include:
   - Location of unit, integration, and E2E tests for the component.
   - Specific setup conditions, mock helpers, and database migrations involved.
   - Example command executions (e.g., running a specific test function).

### Step 4: Run Automated & Manual Test Cases (Interactive Verification)
1. **Run Automated Test Suites**: Run the automated E2E/integration suites first:
   ```bash
   make test-integration
   make test-contract
   make test-e2e
   ```
2. **Active Interaction with Deployed Artifacts**: For validation steps not covered by automated suites, or to perform deeper integration sanity checks:
   - **REST API Interaction**: Trigger endpoints using `curl` against components (e.g., querying Zitadel-protected routes, posting community configurations, and triggering agent assignment reconciliations). Validate HTTP status codes and response schemas.
   - **NATS Message Interaction**: Use NATS clients or scripts to publish event triggers (e.g., dispatching non-conversational commands, simulating hub coordinator requests) and subscribe to specific subjects to verify proper inter-agent routing, topic namespaces, and delivery.
3. **Monitor Cluster Resources & Logs**: Evaluate overall system effectiveness and runtime health:
   - **Structured Logs Inspection**: Stream and inspect active container logs (`kubectl logs -n tacito -l app=<component> --tail=100`) to confirm that logs are generated in structured JSON format, respect observability guidelines, and trace IDs propagate cleanly between components.
   - **Resource Performance and Stability**: Check pod resource usage (`kubectl top pods -n tacito`), look for restart cycles or crash loop conditions, and inspect container events (`kubectl describe pod -n tacito`) to ensure no resource leakage or silent OOM kills are happening under test load.
   
   *(Note: Subagents can be delegated to run the curl requests, publish NATS events, check logs, or monitor resource stats in the background, keeping the main thread free and context light).*

### Step 5: Compile the Verification Report (Concise)
Create a markdown report under the reports directory: `specs/reports/M<Milestone>-verification-report.md`. Keep the report token-efficient:
1. **Milestone Summary**: Target milestone ID, name, list of evaluated specifications, and overall verification result (`PASSED` or `FAILED`).
2. **Checklist Format**: Write a compact checklist representing spec-by-spec status:
   - `[x] SPEC-FR-M6.5.1: Agent Role as Community Assignment Behavior (PASSED)`
3. **Summarized Findings**: 
   - Do not paste raw, long stdout log dumps. 
   - Briefly describe the verification evidence (e.g., "Verified dynamic system prompt construction by checking prompt configmap; details in logs").
   - Include only the specific failure lines/traces for failing specs.

### Step 6: Spec & Milestone Promotion
If all evaluated specifications satisfy their respective Acceptance Criteria:
1. **Promote Spec Status**: Recommend updating the status of each functional specification under the milestone from `IMPLEMENTED` to `VERIFIED`.
2. **Complete the Milestone**: Recommend updating the Milestone status to `🎉 COMPLETED` in the [Specs Index](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/INDEX.md) and its milestone summary file.