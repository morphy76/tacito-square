# TASK-M2.9.1: Documentation validation (RED)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M2.9.1                                 |
| Status        | TODO                                        |
| Spec          | SPEC-FR-M2.9                                |
| Phase         | RED                                         |
| Depends On    | none                                        |

## Description

Write a validation script that verifies all required documentation files exist and contain key sections.

## Work Items

1. Create `test/docs/test_documentation.sh` with checks:
   - `README.md` exists and contains: architecture summary, prerequisites, build instructions, local dev workflow.
   - `tools/helm/tacito-square-infra/README.md` exists (already created in M1).
   - `tools/helm/tacito-square/README.md` exists and documents binding interfaces (no infra sub-chart references).
2. Run the script — assess current compliance.

## Acceptance Criteria

1. `test/docs/test_documentation.sh` exists and is executable.
2. Script validates documentation completeness.
