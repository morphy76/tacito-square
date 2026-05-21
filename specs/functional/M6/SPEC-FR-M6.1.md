# SPEC-FR-M6.1: Community Topology (Hub-Spoke)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M6.1                                |
| Status        | DRAFT                                       |
| Milestone     | M6                                          |
| Component     | keeper, agent                               |
| Depends On    | SPEC-FR-M3.2                                |
| Supersedes    | none                                        |

## Context

Communities organize agents in topologies. Hub-spoke is the initial supported topology where one agent acts as the hub (entry point and router) and other agents are spokes (specialists).

## Specification

1. The system MUST support hub-spoke topology as the default community topology.
2. Each community MUST have exactly one hub agent designated in the TacitoCommunity CRD.
3. The hub agent MUST receive all inbound messages and route them to appropriate spoke agents.
4. Spoke agents MUST respond back through the hub.
5. Topology configuration MUST be stored in the community domain model and reflected in the CRD.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
