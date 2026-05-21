# TASK-M1.2.1: Makefile infrastructure target tests (RED)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M1.2.1                                 |
| Status        | TODO                                        |
| Spec          | SPEC-FR-M1.2                                |
| Phase         | RED                                         |
| Depends On    | none                                        |

## Description

Write a validation script that verifies the expected Makefile infrastructure targets exist and produce correct behavior. This script will fail initially — establishing the RED phase.

## Work Items

1. Create `test/make/test_infra_targets.sh` with the following checks:
   - `make help` output contains `helm-infra-deps`.
   - `make help` output contains `helm-infra-lint`.
   - `make help` output contains `helm-infra-template`.
   - `make help` output contains `helm-infra-install`.
   - `make help` output contains `helm-infra-uninstall`.
   - `make -n helm-infra-lint` succeeds (dry run — validates target exists).
   - `make -n helm-infra-template` succeeds (dry run).
2. Run the script — it MUST FAIL (targets don't exist yet).

## Acceptance Criteria

1. `test/make/test_infra_targets.sh` exists and is executable.
2. Running the script produces clear FAIL output for each missing target.
