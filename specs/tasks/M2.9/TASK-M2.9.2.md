# TASK-M2.9.2: Write project documentation (GREEN)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M2.9.2                                 |
| Status        | TODO                                        |
| Spec          | SPEC-FR-M2.9                                |
| Phase         | GREEN                                       |
| Depends On    | TASK-M2.9.1                                 |

## Description

Write/update README files for the project root and the application Helm chart.

## Work Items

1. Update `README.md` with:
   - Project overview and architecture (4 components, 2 charts).
   - Prerequisites (Go 1.26, Docker, Helm, minikube).
   - Build instructions (`make build`, `make test`, `make docker-build`).
   - Local dev workflow (infra chart → app chart → verify health endpoints).
2. Update `tools/helm/tacito-square/README.md` with:
   - Components included (keeper, agent, operator, bff).
   - Binding interface documentation (how TS_* env vars connect to infra).
   - Installation instructions (prerequisite: infra chart).
   - No references to infrastructure sub-charts.
3. Run `test/docs/test_documentation.sh` — all checks MUST pass.

## Acceptance Criteria

1. All README files contain required sections.
2. All validation checks pass.
