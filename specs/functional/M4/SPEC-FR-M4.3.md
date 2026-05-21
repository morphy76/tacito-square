# SPEC-FR-M4.3: Reconciliation Controller

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M4.3                                |
| Status        | DRAFT                                       |
| Milestone     | M4                                          |
| Component     | operator                                    |
| Depends On    | SPEC-FR-M4.1                                |
| Supersedes    | none                                        |

## Context

The operator must watch TacitoAgent CRDs and reconcile them into Kubernetes Deployments and Services. This is the core control loop that translates agent definitions into running pods.

## Specification

1. The system MUST implement a controller-runtime reconciler for `TacitoAgent` resources.
2. On CRD creation, the reconciler MUST create a Deployment and a headless Service for the agent.
3. The Deployment MUST use the agent Docker image with configuration injected via environment variables derived from the CRD spec.
4. On CRD deletion, the reconciler MUST delete the associated Deployment and Service (using finalizers for cleanup).
5. On CRD update, the reconciler MUST update the Deployment spec (rolling update strategy).
6. The reconciler MUST update TacitoAgent status conditions after each reconciliation.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
