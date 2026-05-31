# TASK-M5.7.5: Add Standalone Agent Helm Makefile Targets

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.7.5                                 |
| Status        | VERIFIED                                    |
| Spec          | SPEC-FR-M5.7                                |
| Depends On    | TASK-M5.7.4                                 |

## Description

Integrate the standalone agent Helm chart into the monorepo root-level Makefile. Define standardized `.PHONY` targets to template, install, uninstall, and execute the automated Helm dry-run test suite for the standalone agent.

## Work Items

1. **RED Phase**:
   - Verify that running `make helm-template-agent` or `make test-helm-agent` fails (or command not found) since the targets do not exist in the root Makefile.
   - Run a check command to verify their absence.

2. **GREEN Phase**:
   - Modify the root `Makefile` to define:
     - `HELM_AGENT_CHART := tools/helm/tacito-agent`
     - `HELM_AGENT_RELEASE ?= ts-agent`
     - `.PHONY` targets: `helm-template-agent`, `helm-install-agent`, `helm-uninstall-agent`, `test-helm-agent`.
     - Document targets with double hash comments (`## `) for automatic help indexing.
   - Run `make test-helm-agent` and verify the test suite executes successfully and passes (GREEN).
   - Run `make helm-template-agent` to verify local rendering works.

3. **REFACTOR Phase**:
   - Ensure alphabetical or logical grouping of new targets in the root `Makefile`.
   - Maintain clean `.PHONY` declarations for all added targets.

## Acceptance Criteria

1. Running `make test-helm-agent` triggers the test suite with exit code `0`.
2. Running `make help` displays descriptions for all new standalone agent Helm targets.
