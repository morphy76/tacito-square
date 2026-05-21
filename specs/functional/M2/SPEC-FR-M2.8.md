# SPEC-FR-M2.8: Continuous Integration (GitHub Actions)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M2.8                                |
| Status        | DRAFT                                       |
| Milestone     | M2                                          |
| Component     | build                                       |
| Depends On    | SPEC-FR-M2.7                                |
| Supersedes    | none                                        |

## Context

Continuous integration ensures code quality on every commit and pull request. The CI pipeline runs tests, linting, and builds automatically via GitHub Actions.

## Specification

1. The system MUST include a GitHub Actions workflow at `.github/workflows/ci.yml`.
2. The workflow MUST trigger on pushes to `main` and on all pull requests.
3. The workflow MUST execute the following steps in order:
   - Checkout code
   - Setup Go 1.26
   - Run `make lint`
   - Run `make test`
   - Run `make build`
   - Run `make docker-build`
4. The system MUST include a Dependabot configuration at `.github/dependabot.yml` for Go module and GitHub Actions version updates.
5. The workflow SHOULD cache Go modules and build artifacts for faster runs.

## Acceptance Criteria

1. `.github/workflows/ci.yml` exists and is valid YAML.
2. `.github/dependabot.yml` exists and is valid YAML.
3. CI workflow runs successfully on push to `main`.
4. CI workflow fails on lint errors or test failures.

## Test Plan

1. Validate workflow syntax with `actionlint .github/workflows/ci.yml`.
2. Review Dependabot config for correct ecosystem (`gomod`, `github-actions`).
3. Push a commit and verify workflow triggers and passes.

## Files Affected

- `.github/workflows/ci.yml` (NEW or MODIFY)
- `.github/dependabot.yml` (NEW or MODIFY)
