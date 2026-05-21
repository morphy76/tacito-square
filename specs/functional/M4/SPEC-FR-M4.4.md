# SPEC-FR-M4.4: Zero-Scaling Support

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M4.4                                |
| Status        | DRAFT                                       |
| Milestone     | M4                                          |
| Component     | operator                                    |
| Depends On    | SPEC-FR-M4.3                                |
| Supersedes    | none                                        |

## Context

Agents should scale to zero when idle and scale up when needed. For minikube environments, operator-managed replica count is the simplest and most reliable approach (no KEDA or custom metrics server required).

## Specification

1. The operator MUST set Deployment replicas to 0 when an agent has been idle for a configurable timeout (default: 5 minutes).
2. The operator MUST scale Deployment replicas to 1 when a NATS message is pending for the agent (via NATS monitoring or keeper notification).
3. Idle detection MUST be based on the `lastHeartbeat` field in the TacitoAgent status.
4. The scaling behavior MUST be configurable per-agent via the TacitoAgent CRD spec (minReplicas, maxReplicas, idleTimeoutSeconds).
5. Scale-up latency SHOULD be under 30 seconds for minikube environments.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
