# Milestone M9: Federation & Hardening

| Field      | Value |
|------------|-------|
| Status     | ⬜ PLANNED |

## Goal

External A2A federation, full observability, comprehensive testing, CI/CD pipeline, production documentation.

## Deliverable

External agent federation via A2A protocol. Prometheus metrics, OpenAPI contracts, E2E tests, benchmark baselines, automated CI/CD, production runbook.

## Specs Required

| Spec ID | Title | FR Ref | Component | Depends On |
|---------|-------|--------|-----------|------------|
| SPEC-FR-06.3 | A2A HTTP gateway | FR-06.3 | keeper | SPEC-FR-06.1 |
| SPEC-FR-06.5 | External source registry | FR-06.5 | keeper | — |
| SPEC-FR-06.6 | External agent messaging | FR-06.6 | keeper | SPEC-FR-06.5 |
| SPEC-FR-06.7 | External source health | FR-06.7 | keeper | SPEC-FR-06.5 |
| SPEC-NFR-METRICS | Prometheus metrics endpoints | NFR-METRICS | all | — |
| SPEC-NFR-OPENAPI | Live OpenAPI spec endpoints | NFR-OPENAPI | all | — |
| SPEC-FR-09.3 | Prometheus metrics | FR-09.3 | all | — |
| SPEC-FR-10.4 | E2E tests | FR-10.4 | test | — |
| SPEC-FR-10.5 | Benchmark tests | FR-10.5 | all | — |
| SPEC-FR-10.6 | Concurrency tests | FR-10.6 | all | — |
| SPEC-FR-12.4 | OpenAPI contracts | FR-12.4 | all | — |
| SPEC-FR-12.5 | Contract tests | FR-12.5 | all | — |
| SPEC-FR-M9-CICD | CI/CD pipeline | — | devops | — |
| SPEC-FR-M9-PROD | Production Helm refinement | — | deploy | — |
| SPEC-FR-M9-TOPO | Mesh & Pipeline topologies | FR-05.2 | agent | SPEC-FR-05.2 |
