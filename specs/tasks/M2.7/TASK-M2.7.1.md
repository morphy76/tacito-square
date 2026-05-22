# TASK-M2.7.1: Dockerfile validation script (RED)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M2.7.1                                 |
| Status        | DONE                                        |
| Spec          | SPEC-FR-M2.7                                |
| Phase         | RED                                         |
| Depends On    | none                                        |

## Description

Write a validation script that verifies each component's Dockerfile conforms to spec: multi-stage build, distroless runtime base, nonroot user, ldflags for size, CGO_ENABLED=0.

## Work Items

1. Create `test/docker/test_dockerfiles.sh` with the following checks per component (keeper, agent, operator, bff):
   - Dockerfile exists at `tools/docker/Dockerfile.<component>`.
   - Build stage uses `golang:` base image.
   - Build stage sets `CGO_ENABLED=0`.
   - Build stage uses `-ldflags="-s -w"`.
   - Runtime stage uses `gcr.io/distroless/base-nossl-debian13:nonroot`.
   - Final `USER` is `nonroot` or runtime base implies nonroot.
2. Run the script — assess current compliance.

## Acceptance Criteria

1. `test/docker/test_dockerfiles.sh` exists and is executable.
2. Script validates all 4 Dockerfiles against spec requirements.
