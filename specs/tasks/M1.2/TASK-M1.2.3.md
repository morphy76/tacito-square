# TASK-M1.2.3: Validate and refactor Makefile (REFACTOR)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M1.2.3                                 |
| Status        | TODO                                        |
| Spec          | SPEC-FR-M1.2                                |
| Phase         | REFACTOR                                    |
| Depends On    | TASK-M1.2.2                                 |

## Description

Run the full validation, clean up Makefile organization, and ensure infrastructure targets are properly grouped and documented.

## Work Items

1. Run `test/make/test_infra_targets.sh` — all checks MUST pass.
2. Organize Makefile sections:
   - Ensure infra targets are grouped under a `## —— Helm (infra) ——` section header.
   - Ensure existing app helm targets are under a `## —— Helm (app) ——` section header.
3. Verify `make helm-infra-lint` executes successfully against the infra chart.
4. Verify `make helm-infra-template` renders valid YAML.
5. Review target ordering for logical flow (deps → lint → template → install → uninstall).

## Acceptance Criteria

1. `test/make/test_infra_targets.sh` passes with all checks GREEN.
2. Makefile has clearly separated sections for app and infra helm targets.
3. `make help` output is well-organized with all targets listed.
