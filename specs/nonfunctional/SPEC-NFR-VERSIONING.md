# SPEC-NFR-VERSIONING: Component Versioning & Lifecycle

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-NFR-VERSIONING                         |
| Status        | DRAFT                                       |
| Component     | all                                         |
| Depends On    | none                                        |

## Context

Tacito Square is a monorepo hosting 4 deployable components and 2 Helm charts. Each component has an independent version lifecycle to enable independent releases, while the system version is represented by the application Helm chart version.

## Specification

1. Each component MUST maintain its own version in a `VERSION.<component>` file at the repository root (e.g., `VERSION.keeper`, `VERSION.agent`, `VERSION.operator`, `VERSION.bff`).
2. Version format MUST follow Semantic Versioning 2.0.0 (`MAJOR.MINOR.PATCH`).
3. The application Helm chart version (`Chart.yaml` `version` field) represents the **system version**.
4. The infrastructure Helm chart has its own independent version.
5. Git tags MUST follow the format `<component>-v<version>` (e.g., `keeper-v0.2.0`, `agent-v0.1.3`).
6. Helm chart tags MUST follow `chart-<chart-name>-v<version>` (e.g., `chart-tacito-square-v0.2.0`).
7. Docker image tags MUST match the component version from `VERSION.<component>`.
8. Version bumps MUST be atomic (one commit per version change).

## Acceptance Criteria

1. Each `VERSION.<component>` file exists and contains a valid semver string.
2. `make docker-build` tags images with the correct component version.
3. Git tags follow the `<component>-v<version>` convention.
4. Chart versions in `Chart.yaml` are independent from component versions.

## Test Plan

To be verified during M2 execution.

## Files Affected

- `VERSION.keeper`, `VERSION.agent`, `VERSION.operator`, `VERSION.bff`
- `tools/helm/tacito-square/Chart.yaml`
- `tools/helm/tacito-square-infra/Chart.yaml`
- `Makefile`
