# TASK-M1.1.5: Validate and refactor (REFACTOR)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M1.1.5                                 |
| Status        | TODO                                        |
| Spec          | SPEC-FR-M1.1                                |
| Phase         | REFACTOR                                    |
| Depends On    | TASK-M1.1.4                                 |

## Description

Run the full validation script from TASK-M1.1.1 and ensure all checks pass. Clean up values, ensure consistent naming conventions, and add chart documentation.

## Work Items

1. Run `test/helm/test_infra_chart.sh` — all checks MUST pass (GREEN → REFACTOR).
2. Review `values.yaml` for consistent naming and sensible grouping.
3. Ensure `NOTES.txt` provides useful post-install information (service endpoints, credentials).
4. Add a `README.md` to `tools/helm/tacito-square-infra/` with:
   - What services are included
   - Quick install instructions
   - Key configuration knobs
5. Verify conditional toggling: for each of the 7 sub-charts, confirm `--set <component>.enabled=false` cleanly excludes it.

## Acceptance Criteria

1. `test/helm/test_infra_chart.sh` passes with all checks GREEN.
2. `tools/helm/tacito-square-infra/README.md` exists with install instructions.
3. Each sub-chart can be independently disabled without errors.
4. No warnings from `helm lint`.
