# TASK-M2-025

| Field       | Value |
|-------------|-------|
| Task ID     | TASK-M2-025 |
| Spec        | SPEC-FR-10.7 |
| Milestone   | M2 |
| Status      | ⬜ Pending |

## Description

Introduce `.github/` common GitHub Actions workflows to handle component lifecycle automation:

- **CI on pull requests**: run `make test` for each component on every PR targeting `main` or release branches.
- **CI on merge / push to main**: full build + test across all components (`agent`, `keeper`, `bff`, `operator`).
- **Dependabot periodic updates**: `.github/dependabot.yml` configured for `gomod` and `github-actions` ecosystems with weekly schedule.
- **Static code scan**: integrate `golangci-lint` (with `staticcheck`, `gosec`, `errcheck`) as a required workflow step on every PR.

## Files

- `.github/workflows/ci.yml` — PR and push-to-main pipeline (test, build, lint)
- `.github/workflows/scan.yml` — static analysis and security scan (`golangci-lint` + `gosec`)
- `.github/dependabot.yml` — automated dependency update schedule for `go.mod` and Actions
- `.github/workflows/release.yml` — optional: tag-triggered image build per component (stubs `VERSION.*` files)
