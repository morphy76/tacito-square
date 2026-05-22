# Milestone M1: Infrastructure Helm Chart

| Field      | Value |
|------------|-------|
| Status     | ✔️ IMPLEMENTED |

## Goal

Create a dedicated infrastructure Helm chart that delivers all external dependencies (NATS, Redis, PostgreSQL, Qdrant, OTel Collector, Keycloak, MinIO) as a separate artifact from the application chart. This enables clean separation between infrastructure and application concerns.

## Deliverable

`helm install tacito-infra tools/helm/tacito-square-infra/` → all infrastructure services running and accessible. Makefile targets for infrastructure lifecycle.

## Specs Required

| Spec ID | Title | Component | Depends On |
|---------|-------|-----------|------------|
| SPEC-FR-M1.1 | Infrastructure Helm Chart | deploy | none |
| SPEC-FR-M1.2 | Makefile Infrastructure Targets | build | SPEC-FR-M1.1 |
