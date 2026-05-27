# TASK-M4.6.3: Outbound Progression NATS Event Telemetry

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M4.6.3                                 |
| Status        | IMPLEMENTED                                 |
| Spec          | SPEC-FR-M4.6                                |
| Depends On    | TASK-M4.6.1                                 |

## Description

Design and implement NATS event broadcasting inside the `K8sCRDCoordinator` background loop. The coordinator must publish structured JSON progression events (`started`, `completed`, `failed` with error logs) to subjects `agent.provisioning.started`, `agent.provisioning.completed`, and `agent.provisioning.failed`, and wire up the real coordinator inside Keeper bootstrap logic.

## Boundary & Target Functions

- **Package**: `internal/keeper/adapters/outbound/crd` & `internal/keeper`
- **Files**:
  - `internal/keeper/adapters/outbound/crd/crd_coordinator.go`
  - `internal/keeper/bootstrap.go`
- **Target Functions**:
  - `(c *K8sCRDCoordinator) PublishProvisioningEvent(ctx context.Context, subject string, agent *model.Agent, errVal error)`

## Work Items

1. **RED Phase**:
   * Implement unit tests using mock NATS connections to assert:
     * A structured event is published to `agent.provisioning.started` at goroutine start.
     * `agent.provisioning.completed` is published on successful Kube-API write.
     * `agent.provisioning.failed` carrying the exact error message is published if Kube-API fails after all retries.

2. **GREEN Phase**:
   * Inject NATS connection (`*nats.Conn`) into the `K8sCRDCoordinator` constructor.
   * Implement event serialization (`tenant_id`, `agent_id`, `community_id`, `timestamp`, `error`).
   * Inside Keeper's `bootstrap.go`, replace `noOpCRDCoordinator` with the real `K8sCRDCoordinator`, configuring it with the real Kube and NATS handles.

3. **REFACTOR Phase**:
   * Verify all payload logs contain the correct tracing (`trace_id` and `span_id`) extracted from context.

## Acceptance Criteria

1. Progress signalling integration tests pass under simulated write failures.
2. Structured JSON events conform to the contract payloads in the specification.
