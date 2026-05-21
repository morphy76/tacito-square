# SPEC-FR-M4.1: Agent CRD Definition & Registration

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M4.1                                |
| Status        | DRAFT                                       |
| Milestone     | M4                                          |
| Component     | operator                                    |
| Depends On    | none                                        |
| Supersedes    | none                                        |

## Context

The operator needs CRD definitions for TacitoAgent resources in the `tacito.square.io/v1alpha1` API group. This is the foundational resource that represents an agent instance in the cluster.

## Specification

1. The system MUST define a `TacitoAgent` CRD in API group `tacito.square.io/v1alpha1`.
2. The CRD spec MUST include fields: agentName, communityRef, llmConfig (model, temperature, maxTokens), systemPrompt, resources (cpu/memory requests and limits), replicas.
3. The CRD status MUST include fields: phase (Pending, Running, Idle, Terminated), conditions (standard K8s conditions), lastHeartbeat.
4. The CRD MUST be generated using Kubebuilder markers (per SPEC-NFR-STACK).
5. CRD installation MUST be handled by the application Helm chart.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
