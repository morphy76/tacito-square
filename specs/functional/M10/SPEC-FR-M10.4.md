# SPEC-FR-M10.4: Production Helm & Hardening

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M10.4                                |
| Status        | DRAFT                                       |
| Milestone     | M10                                          |
| Component     | deploy                                      |
| Depends On    | SPEC-FR-M2.1                                |
| Supersedes    | none                                        |

## Context

Production deployments require TLS, secrets management, HA, and security hardening.

## Specification

1. Production Helm values MUST configure TLS termination.
2. Secrets MUST be managed via K8s Secrets or external secrets operator.
3. All components MUST run with read-only root filesystem and non-root user.
4. Pod security standards MUST be enforced (restricted profile).
5. Resource limits MUST be tuned with documented baselines.
6. A production runbook MUST cover deployment, rollback, and incident response.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
