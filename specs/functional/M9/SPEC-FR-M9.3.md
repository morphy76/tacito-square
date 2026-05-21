# SPEC-FR-M9.3: Prometheus Metrics Integration

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M9.3                                |
| Status        | DRAFT                                       |
| Milestone     | M9                                          |
| Component     | all                                         |
| Depends On    | SPEC-NFR-METRICS                            |
| Supersedes    | none                                        |

## Context

All components expose Prometheus metrics for operational monitoring beyond HTTP auto-instrumentation.

## Specification

1. Keeper: `tacito_agents_total` (gauge), `tacito_communities_total` (gauge), `tacito_hitl_pending` (gauge).
2. Agent: `tacito_agent_messages_total` (counter), `tacito_agent_llm_tokens_total` (counter), `tacito_agent_reasoning_duration_seconds` (histogram).
3. Operator: `tacito_reconciliations_total` (counter), `tacito_agent_pods_total` (gauge).
4. All metrics MUST be registered via Prometheus Go client.
5. Helm charts MUST include ServiceMonitor CRDs.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
