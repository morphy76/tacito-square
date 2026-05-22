# TASK-M2.9.3: Write infrastructure documentation (GREEN)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M2.9.3                                 |
| Status        | DONE                                        |
| Spec          | SPEC-FR-M2.9                                |
| Phase         | GREEN                                       |
| Depends On    | TASK-M2.9.1                                 |

## Description

Write/update README file for the infrastructure Helm chart.

## Work Items

1. Update `tools/helm/tacito-square-infra/README.md` with:
   - What services are included.
   - Installation instructions.
   - Configuration reference (key values.yaml knobs).
2. Run `test/docs/test_documentation.sh` — all checks MUST pass.

## Acceptance Criteria

1. The infrastructure chart README file contains required sections.
2. All validation checks pass.
