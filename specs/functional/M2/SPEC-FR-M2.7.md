# SPEC-FR-M2.7: Container Images (distroless, multi-stage)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M2.7                                |
| Status        | DRAFT                                       |
| Milestone     | M2                                          |
| Component     | build                                       |
| Depends On    | SPEC-FR-M2.3, SPEC-FR-M2.4, SPEC-FR-M2.5, SPEC-FR-M2.6 |
| Supersedes    | none                                        |

## Context

Each component is packaged as a minimal Docker image using multi-stage builds with distroless runtime bases. Images must not contain source code, build tools, or unnecessary OS packages.

## Specification

1. Each component MUST have a Dockerfile at `tools/docker/Dockerfile.<component>`.
2. Build stage MUST use `golang:1.26-alpine` with `CGO_ENABLED=0`.
3. Runtime stage MUST use `gcr.io/distroless/base-nossl-debian13:nonroot`.
4. The binary MUST be compiled with `-ldflags="-s -w"` for minimal size.
5. The runtime container MUST run as `nonroot:nonroot` user.
6. `make docker-build` MUST build all 4 component images.
7. `make docker-build-<component>` MUST build individual component images.
8. Image tags MUST use the version from `VERSION.<component>` files.
9. The `REGISTRY` variable MUST be configurable (default: `localhost:5000/tacito-square`).

## Acceptance Criteria

1. `make docker-build` succeeds for all 4 components.
2. `docker images | grep tacito-square` shows all 4 images.
3. Each image runs as non-root user (verified via `docker inspect`).
4. No image contains Go toolchain or source code.
5. Each image is under 30MB (distroless + static binary).

## Test Plan

1. `make docker-build` — exit code 0 for all components.
2. `docker run --rm <image> /healthz` — verify binary starts (expect connection error since no infra, but binary runs).
3. `docker inspect <image> --format '{{.Config.User}}'` — returns `nonroot`.
4. `docker history <image>` — verify 2-stage build (no Go layers in final image).

## Files Affected

- `tools/docker/Dockerfile.agent` (VERIFY — matches spec)
- `tools/docker/Dockerfile.keeper` (VERIFY — matches spec)
- `tools/docker/Dockerfile.operator` (VERIFY — matches spec)
- `tools/docker/Dockerfile.bff` (VERIFY — matches spec)
- `Makefile` (VERIFY — docker targets)
