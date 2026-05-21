# SPEC-FR-M4.2: AgentCommunity CRD Definition

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M4.2                                |
| Status        | DRAFT                                       |
| Milestone     | M4                                          |
| Component     | operator                                    |
| Depends On    | SPEC-FR-M4.1                                |
| Supersedes    | none                                        |

## Context

The TacitoCommunity CRD represents a community of agents in the cluster. It defines the topology, member agents, and community-level configuration that the operator uses for orchestration.

## Specification

1. The system MUST define a `TacitoCommunity` CRD in API group `tacito.square.io/v1alpha1`.
2. The CRD spec MUST include fields: communityName, topology (enum: hub-spoke), agentRefs (list of TacitoAgent references), quotas (maxAgents, maxTokens), natsNamespace.
3. The CRD status MUST include fields: phase (Pending, Active, Suspended, Terminated), activeAgents (count), conditions.
4. The CRD MUST be generated using Kubebuilder markers.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
