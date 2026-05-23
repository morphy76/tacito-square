# SPEC-FR-M4.6: Agent CRD Submission

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M4.6                                |
| Status        | DRAFT                                       |
| Milestone     | M4                                          |
| Component     | keeper                                      |
| Depends On    | SPEC-FR-M3.7, SPEC-FR-M4.1                 |
| Supersedes    | none                                        |

## Context

When an agent is assigned to a community, keeper constructs and submits a TacitoAgent CRD to the K8s API. The operator (M4) will then reconcile this CRD into a running agent pod.

## Specification

1. The system MUST construct a `TacitoAgent` CR (apiVersion: `tacito.square.io/v1alpha1`) from the agent definition and community configuration.
2. The system MUST use `client-go` to submit the CR to the K8s API server.
3. The CR spec MUST include: agent name, LLM config, system prompt, community reference, resource limits.
4. The system MUST handle submission failures gracefully (conflict, validation, connectivity) with retries and exponential backoff (per SPEC-NFR-CLOUD).
5. The system MUST update the agent-community assignment status based on CRD submission result.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
