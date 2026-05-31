# TASK-M5.4.2: Qdrant Long-Term Memory Infrastructure Adapter

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.4.2                                 |
| Status        | VERIFIED                                    |
| Spec          | SPEC-FR-M5.4                                |
| Depends On    | TASK-M5.4.1                                 |

## Description

Implement the concrete Qdrant infrastructure adapter (`QdrantLTMAdapter`) in the adapters layer. It must implement the `LongTermMemory` outbound port, connect over gRPC using the approved Qdrant Go client, enforce strict multi-tenant payload filters for visibility and isolation, and integrate standard OpenTelemetry tracing.

## Work Items

1. **RED Phase**:
   - Write integration tests inside `internal/agent/adapters/outbound/qdrant/ltm_adapter_test.go` using `testcontainers-go` and `testcontainers/qdrant` that assert:
     - Strict multi-tenant data isolation (tenant A searches cannot leak tenant B points).
     - Scoped visibility filters (private vs. community vs. tenant visibility access logic).
     - Storage of points containing the exact vector and JSON metadata payload.
     - Outbound OTel tracing span coverage and Prometheus metric latency records.
   - Run tests and witness failure (RED).

2. **GREEN Phase**:
   - Create `internal/agent/adapters/outbound/qdrant/ltm_adapter.go`.
   - Implement `QdrantLTMAdapter` using the `github.com/qdrant/go-client/grpc` client library.
   - Enforce mandatory payload filters inside the gRPC `SearchPoints` and `UpsertPoints` constructs (matching the `tenant_id` and visibility matrix rules).
   - Wire standard OpenTelemetry instrumentation in the Qdrant client commands.
   - Run the integration tests and verify they pass successfully (GREEN).

3. **REFACTOR Phase**:
   - Review Qdrant client connection pooling, timeouts, and ensure Go context propagation is actively handled across all network interactions.

## Acceptance Criteria

1. Qdrant adapter complies with the `LongTermMemory` outbound port contract.
2. The search and save commands always inject the mandatory `tenant_id` and permission scope conditions.
3. Tracing spans are correctly generated and propagated on Qdrant interactions.
