# TASK-M8.10.1: CI workflow validation (RED)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M8.10.1                                |
| Status        | TODO                                        |
| Spec          | SPEC-FR-M8.10                               |
| Phase         | RED                                         |
| Depends On    | none                                        |

## Description

Write a validation script that verifies the CI workflow file exists and contains expected steps.

## Work Items

1. Create `test/ci/test_ci_workflow.sh` with the following checks:
   - `.github/workflows/ci.yml` exists.
   - Workflow triggers on `push` to `main` and on `pull_request`.
   - Workflow includes steps: checkout, Go setup, `make lint`, `make test`, `make build`, `make docker-build`.
   - `.github/dependabot.yml` exists.
   - Dependabot config covers `gomod` and `github-actions` ecosystems.
2. Run the script — it MUST FAIL (files don't exist yet).

## Acceptance Criteria

1. `test/ci/test_ci_workflow.sh` exists and is executable.
2. Running the script produces clear FAIL output.
