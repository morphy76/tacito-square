# SPEC-FR-M2.9: Project Documentation

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M2.9                                |
| Status        | DRAFT                                       |
| Milestone     | M2                                          |
| Component     | docs                                        |
| Depends On    | all M2 specs                                |
| Supersedes    | none                                        |

## Context

Clear documentation enables developers to build, test, and deploy the project. Both the project root README and the Helm chart READMEs must be up-to-date with the two-chart architecture.

## Specification

1. The project `README.md` MUST include:
   - Project overview and architecture summary.
   - Prerequisites (Go 1.26, Docker, Helm, minikube/Kind).
   - Build instructions (`make build`, `make test`, `make docker-build`).
   - Local development workflow (infra chart → app chart → verify).
2. The infrastructure chart MUST have a `README.md` at `tools/helm/tacito-square-infra/README.md` with:
   - What services are included.
   - Installation instructions.
   - Configuration reference (key values.yaml knobs).
3. The application chart MUST have an updated `README.md` at `tools/helm/tacito-square/README.md` with:
   - What components are included.
   - Binding interface documentation (how to configure infra connections).
   - Installation instructions (prerequisite: infra chart).

## Acceptance Criteria

1. `README.md` contains accurate build and run instructions.
2. `tools/helm/tacito-square-infra/README.md` exists with installation and configuration docs.
3. `tools/helm/tacito-square/README.md` documents binding interfaces and does not reference infrastructure sub-charts.

## Test Plan

1. Review README files for completeness and accuracy.
2. Follow README instructions on a clean environment — verify they work.

## Files Affected

- `README.md` (MODIFY)
- `tools/helm/tacito-square-infra/README.md` (NEW)
- `tools/helm/tacito-square/README.md` (MODIFY)
