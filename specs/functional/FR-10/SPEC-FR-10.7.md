# SPEC-FR-10.7: Makefile Targets

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-FR-10.7                       |
| Status        | VERIFIED                           |
| Milestone     | M1                                 |
| FR/NFR Ref    | FR-10.7                            |
| Component     | all                                |
| Depends On    | —                                  |

## Context

The Makefile is the single entry point for all build, test, lint, Docker, and Helm operations.

## Specification

1. The Makefile MUST define the following targets:

### Build
- `all` — lint + test + build
- `build` — all binaries (`cmd/agent`, `cmd/keeper`, `cmd/operator`, `cmd/bff`)
- `build-{component}` — individual component build

### Test
- `test` — unit tests with `-race`
- `test-integration` — testcontainers-based integration tests
- `test-operator` — envtest operator tests
- `test-e2e` — Kind-based E2E tests
- `test-bench` — benchmark tests
- `test-race` — all tests with race detector
- `test-contract` — OpenAPI contract tests

### Quality
- `lint` — golangci-lint
- `generate` — code generation

### Docker
- `docker-build` — build all images
- `docker-build-{component}` — individual image build
- `docker-push` — push all images

### Helm
- `helm-template` — render templates locally
- `helm-install` — install/upgrade release
- `helm-uninstall` — remove release

### CI
- `ci` — full CI pipeline (lint + test + integration + contract + build + docker)

### Utilities
- `clean` — remove build artifacts
- `help` — show targets with descriptions

2. Version files (`VERSION.agent`, etc.) MUST drive image tags.
3. `REGISTRY` MUST default to `localhost:5000/tacito-square`.
4. Every target MUST have a `## description` comment for `make help`.

## Acceptance Criteria

1. `make help` lists all targets with descriptions
2. `make test` runs unit tests with race detection
3. `make docker-build-keeper` builds with correct registry and tag
4. `REGISTRY` override works: `make docker-build REGISTRY=ghcr.io/foo`

## Files

- `Makefile` ✅ IMPLEMENTED (117 lines, 20+ targets)
