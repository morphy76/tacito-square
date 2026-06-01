# Milestone M9: Federation & Hardening

| Field      | Value |
|------------|-------|
| Status     | ⬜ PLANNED |

## Goal

External A2A federation, full observability with Prometheus metrics, OpenAPI contract validation, comprehensive testing (E2E, benchmarks), and production hardening.

## Deliverable

External agent federation via A2A protocol. Prometheus dashboards, validated OpenAPI contracts, E2E test suite on Kind cluster, production-ready Helm values with TLS and secrets management.

## Specs Required

| Spec ID | Title | Component | Depends On |
|---------|-------|-----------|------------|
| SPEC-FR-M9.1 | A2A HTTP Gateway | keeper | SPEC-FR-M6.5 |
| SPEC-FR-M9.2 | External Agent Registry | keeper | SPEC-FR-M9.1 |
| SPEC-FR-M9.3 | Prometheus Metrics Integration | all | SPEC-NFR-OBSERVABILITY |
| SPEC-FR-M9.4 | OpenAPI Contract Validation | all | SPEC-NFR-OPENAPI |
| SPEC-FR-M9.5 | E2E & Benchmark Tests | test | all M1-M8 |
| SPEC-FR-M9.6 | Production Helm & Hardening | deploy | SPEC-FR-M2.1 |
| SPEC-FR-M9.7 | K8s NetworkPolicies | operator | SPEC-FR-M4.3, SPEC-FR-M6.3 |
| SPEC-FR-M9.9 | Benchmark Suite & Integration Coverage Verification | test | none |
