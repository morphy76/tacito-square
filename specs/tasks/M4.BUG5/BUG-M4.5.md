# BUG-M4.5: Echo Endpoint Fails with 503 Due to Static Database Status Check

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M4.5                                                           |
| Status        | OPEN                                                               |
| Severity      | HIGH                                                               |
| Milestone     | M4 — Operator Core                                                 |
| Affects       | internal/keeper/application/service/echo_service.go                 |
| Violates      | SPEC-FR-M4.8                                                       |
| Discovered    | Runtime testing of the community echo endpoint                     |

## Problem Statement

The `EchoServiceImpl` in [echo_service.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/application/service/echo_service.go) filters active agents using a static helper function `filterRunningAgents`. This helper function reads the `Status` property directly from the database-mapped agent model (`a.Status == model.AgentStatusRunning`).

However, the Keeper component never writes or persists the `"running"` status to the database. When an agent is deployed via the lifecycle service, its database status is set to `"pending"`, but the database is never updated to `"running"` when the pod becomes healthy. Instead, the `"running"` status is only calculated dynamically at runtime via the `CRDCoordinator.GetAgentCRDStatus` during `/status` queries.

Because `filterRunningAgents` relies on the static database status, it finds zero running agents, causing the `/echo` endpoint to fail with an HTTP `503 Service Unavailable` error and the message `"no running agents in community"`, even when the corresponding agent pods are fully healthy, live, ready, and running in the Kubernetes cluster.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| keeper / service | [echo_service.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/application/service/echo_service.go) | `filterRunningAgents` filters using static database status `a.Status == model.AgentStatusRunning` which is never set to `"running"` in the database. |

## Impact

1. **Echo Endpoint Failure:** The `/echo` endpoint is completely unusable in active runtime clusters, always returning `503 Service Unavailable` even when the target community and agents are fully operational.
2. **Connectivity Validation Gap:** The key validation pipeline meant to smoke-test Keeper ↔ NATS ↔ Agent connectivity fails by default.

## Expected Behaviour

1. The `EchoServiceImpl` MUST query the real-time status of all assigned agents in parallel or sequence via the `CRDCoordinator` to determine if their actual runtime state is `"running"` (corresponding to `v1alpha1.PhaseRunning`).
2. The endpoint MUST NOT rely solely on the static database status (`a.Status`) to filter active agents for fanning out echo requests.

## Acceptance Criteria

1. Running a community echo request on a community with active running agent pods successfully resolves their status and fans out the requests, returning `200 OK` with decorated replies instead of `503 Service Unavailable`.
2. Existing unit tests in `echo_service_test.go` are updated to assert this real-time dynamic check and pass cleanly.
