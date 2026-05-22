# Milestone M2: Application Helm Chart & Component Scaffolding

| Field      | Value |
|------------|-------|
| Status     | ⬜ PLANNED |

## Goal

Refactor the application Helm chart to be infrastructure-free (binding interfaces only) and scaffold all 4 components (keeper, agent, operator, bff) as hello-world HTTP servers with health probes, structured logging, and OpenTelemetry tracing.

## Deliverable

`helm install tacito tools/helm/tacito-square/` → 4 components deployed, each responding to `/healthz` and `/readyz`, logging JSON to stdout, exporting traces. Docker images built from distroless base. CI pipeline running.

## Specs Required

| Spec ID | Title | Component | Depends On |
|---------|-------|-----------|------------|
| SPEC-FR-M2.1 | Application Helm Chart (infra-free, binding interfaces) | deploy | SPEC-FR-M1.1 |
| SPEC-FR-M2.2 | Shared Foundation Library (config, health, logging, tracing, errors) | shared | none |
| SPEC-FR-M2.3 | Keeper Hello World | keeper | SPEC-FR-M2.2 |
| SPEC-FR-M2.4 | Agent Hello World | agent | SPEC-FR-M2.2 |
| SPEC-FR-M2.5 | Operator Hello World | operator | SPEC-FR-M2.2 |
| SPEC-FR-M2.6 | BFF Hello World | bff | SPEC-FR-M2.2 |
| SPEC-FR-M2.7 | Container Images (distroless, multi-stage) | build | SPEC-FR-M2.3, SPEC-FR-M2.4, SPEC-FR-M2.5, SPEC-FR-M2.6 |
| SPEC-FR-M2.8 | Continuous Integration (GitHub Actions) | build | SPEC-FR-M2.7 |
| SPEC-FR-M2.9 | Project Documentation | docs | all |
| SPEC-FR-M2.10 | Avoid Bitnami (Leverage Free & Non-Commercial Infrastructural Dependencies) | deploy | SPEC-FR-M1.1 |
