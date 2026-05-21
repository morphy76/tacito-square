# SPEC-FR-M9.7: K8s NetworkPolicies

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M9.7                                |
| Status        | DRAFT                                       |
| Milestone     | M9                                          |
| Component     | operator                                    |
| Depends On    | SPEC-FR-M4.3, SPEC-FR-M6.3                 |
| Supersedes    | none                                        |

## Context

Network isolation between communities prevents unauthorized cross-community traffic.

## Specification

1. The operator MUST create NetworkPolicy resources for each TacitoCommunity.
2. NetworkPolicies MUST allow only intra-community pod-to-pod traffic.
3. NetworkPolicies MUST allow traffic from keeper and operator pods (control plane).
4. NetworkPolicies MUST allow egress to infrastructure services.
5. NetworkPolicies MUST deny all other ingress and egress by default.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
