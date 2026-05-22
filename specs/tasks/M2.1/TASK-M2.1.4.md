# TASK-M2.1.4: Validate and refactor (REFACTOR)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M2.1.4                                 |
| Status        | TODO                                        |
| Spec          | SPEC-FR-M2.1                                |
| Phase         | REFACTOR                                    |
| Depends On    | TASK-M2.1.3                                 |

## Description

Run the full validation script and clean up chart templates.

## Work Items

1. Run `test/helm/test_app_chart.sh` — all checks MUST pass.
2. Verify `helm lint` passes with no warnings beyond the icon recommendation.
3. Ensure consistent label selectors across all templates.
4. Verify conditional rendering: `operator.enabled=false` / `bff.enabled=false` cleanly omits those components.

## Acceptance Criteria

1. All validation checks pass.
2. `helm lint` clean.
3. Conditional rendering verified for operator and bff.
