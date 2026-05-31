# TASK-M5.2.3: Message Processing Use Case & Subscriber Integration

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.2.3                                 |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M5.2                                |
| Depends On    | TASK-M5.2.2                                 |

## Description

Create the inbound port and orchestrating application service use case to handle incoming messages and invoke the stateless `Brain` port. Integrate this pipeline into the NATS `echo_subscriber` driving adapter.

## Work Items

1. **RED Phase**:
   - Write a unit test `internal/agent/application/service/message_processor_test.go` asserting that `MessageProcessor` maps an incoming text payload to a stateless `BrainRequest` and invokes the mocked `Brain` port correctly.
   - Write an integration test `internal/agent/adapters/inbound/nats/echo_subscriber_integration_test.go` checking that publishing an echo message triggers the processing service pipeline and replies via NATS.
   - Run tests and witness expected failures (RED).

2. **GREEN Phase**:
   - Define the `MessageProcessor` inbound port interface under `internal/agent/application/ports/inbound/message.go`.
   - Implement `MessageProcessor` service inside `internal/agent/application/service/message_processor.go`.
   - Update `internal/agent/adapters/inbound/nats/echo_subscriber.go` to inject the `MessageProcessor` service, executing it upon event reception and publishing the reasoning result.
   - Update agent bootstrapping in `internal/agent/bootstrap.go` and `cmd/agent/main.go` to construct and wire the use case services.
   - Verify tests pass successfully (GREEN).

3. **REFACTOR Phase**:
   - Clean up bootstrap instantiation scopes and ensure trace context is correctly propagated from NATS headers to the use case context.

## Acceptance Criteria

1. Inbound driving ports and services are fully separated from concrete adapters.
2. The NATS subscription pipeline executes the use case to generate reasoning outputs and return answers.
