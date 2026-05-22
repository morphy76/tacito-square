# SPEC-NFR-BUILDING: Build System

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-NFR-BUILDING                           |
| Status        | ACCEPTED                                    |
| Component     | all                                         |
| Depends On    | SPEC-NFR-VERSIONING                         |

## Context

Tacito Square is a monorepo with 4 deployable components (keeper, agent, operator, bff) and 2 Helm charts (application, infrastructure). A single root Makefile orchestrates all build, test, container, and deployment operations.

## Specification

1. Builds MUST be driven by a single root `Makefile` (not per-component Makefiles).
2. The Makefile MUST provide the following target categories:
   - **Build**: `build`, `build-<component>` — compile Go binaries.
   - **Test**: `test` (unit), `test-integration`, `test-operator`, `test-e2e`, `test-bench`, `test-race`, `test-contract`.
   - **Quality**: `lint`, `generate`.
   - **Docker**: `docker-build`, `docker-build-<component>`, `docker-push`.
   - **Helm (app)**: `helm-template`, `helm-install`, `helm-uninstall`.
   - **Helm (infra)**: `helm-infra-deps`, `helm-infra-lint`, `helm-infra-template`, `helm-infra-install`, `helm-infra-uninstall`.
   - **CI**: `ci` — full pipeline (lint + test + build + docker).
   - **Utilities**: `clean`, `help`.
3. All targets MUST be declared `.PHONY`.
4. Build flags MUST be consistent across all components: `CGO_ENABLED=0`, `-ldflags="-s -w"`.
5. Component versions MUST be read from `VERSION.<component>` files.
6. Container image tags MUST use the component version.
7. The `help` target MUST auto-generate target documentation from `## ` comments.

## Acceptance Criteria

1. `make help` lists all targets with descriptions.
2. `make build` compiles all 4 component binaries to `bin/`.
3. `make test` runs unit tests with race detector.
4. `make docker-build` builds all 4 container images.
5. `make helm-infra-install` deploys infrastructure.
6. `make helm-install` deploys application components.
7. `make ci` runs the full pipeline without errors.

## Test Plan

To be verified during M1 and M2 execution.

## Files Affected

- `Makefile`
