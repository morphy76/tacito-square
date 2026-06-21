# TASK-M9.10.2: Create CI workflow and Dependabot config (GREEN)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M9.10.2                                |
| Status        | TODO                                        |
| Spec          | SPEC-FR-M9.10                               |
| Phase         | GREEN                                       |
| Depends On    | TASK-M9.10.1                                |

## Description

Create GitHub Actions CI workflow and Dependabot configuration.

## Work Items

1. Create `.github/workflows/ci.yml` with:
   - Trigger: push to `main`, all pull requests.
   - Jobs: checkout → setup Go 1.26 → `make lint` → `make test` → `make build` → `make docker-build`.
   - Go module and build cache.
2. Create `.github/dependabot.yml` with ecosystems: `gomod`, `github-actions`.
3. Run `test/ci/test_ci_workflow.sh` — all checks MUST pass.

## Acceptance Criteria

1. Both files exist and are valid YAML.
2. All validation checks pass.
