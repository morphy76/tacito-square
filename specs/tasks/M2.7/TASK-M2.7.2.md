# TASK-M2.7.2: Verify and fix Dockerfiles (GREEN)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M2.7.2                                 |
| Status        | DONE                                        |
| Spec          | SPEC-FR-M2.7                                |
| Phase         | GREEN                                       |
| Depends On    | TASK-M2.7.1, TASK-M2.3.2, TASK-M2.4.2, TASK-M2.5.2, TASK-M2.6.2 |

## Description

Verify each Dockerfile matches the spec. Fix any deviations. Ensure `make docker-build` succeeds for all 4 components (requires hello-world binaries from M2.3–M2.6).

## Work Items

1. Review and fix `tools/docker/Dockerfile.keeper` to match spec.
2. Review and fix `tools/docker/Dockerfile.agent`.
3. Review and fix `tools/docker/Dockerfile.operator`.
4. Review and fix `tools/docker/Dockerfile.bff`.
5. Run `make docker-build` — all 4 images MUST build successfully.
6. Run `test/docker/test_dockerfiles.sh` — all checks MUST pass.

## Acceptance Criteria

1. All 4 Dockerfiles conform to spec (distroless, nonroot, CGO_ENABLED=0, ldflags).
2. `make docker-build` succeeds.
3. All validation checks pass.
