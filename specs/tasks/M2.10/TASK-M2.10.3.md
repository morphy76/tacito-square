# TASK-M2.10.3: Verify chart configuration & consistency (REFACTOR)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M2.10.3                                |
| Status        | TODO                                        |
| Spec          | SPEC-FR-M2.10                               |
| Phase         | REFACTOR                                    |
| Depends On    | TASK-M2.10.2                                |

## Description

Validate and refactor the updated infrastructure and application charts to ensure binding hostnames and variables align perfectly, keeping resources optimized.

## Work Items

1. Run `make helm-infra-lint` to verify updated infrastructure chart has zero lint errors.
2. Run `make helm-infra-template` to ensure all manifests render correctly.
3. Review `values.yaml` in both charts to verify binding hosts (`TS_KEEPER_DB_HOST` etc.) match the new Helm release naming and structures.
4. Clean up any leftover caches or stale dependencies.

## Acceptance Criteria

1. `make helm-infra-lint` passes cleanly with no warnings or errors.
2. Template rendering completes successfully.
3. Documentation references are fully updated.
