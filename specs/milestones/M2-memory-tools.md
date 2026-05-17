# Milestone M2: Memory, Tools, Storage & Cache

| Field      | Value |
|------------|-------|
| Status     | 🔄 IN PROGRESS (20/24 tasks) |
| Tests      | 32 (unit), integration pending |

## Goal

Agents have functional short-term and long-term memory, and can invoke external tools via MCP. Infrastructure supports S3 payload offloading and Redis caching.

## Deliverable

Agent processes multi-turn conversation with memory persistence and tool use.

## Specs Required

| Spec ID | Title | FR Ref | Status | File |
|---------|-------|--------|--------|------|
| SPEC-FR-04.2 | Short-term memory (Redis adapter) | FR-04.2 | ✅ IMPLEMENTED | [SPEC-FR-04.2](../functional/FR-04/SPEC-FR-04.2.md) |
| SPEC-FR-04.3 | Long-term memory (Qdrant adapter) | FR-04.3 | ✅ IMPLEMENTED | [SPEC-FR-04.3](../functional/FR-04/SPEC-FR-04.3.md) |
| SPEC-FR-04.4 | Tool invocation (MCP adapter) | FR-04.4 | ✅ IMPLEMENTED | [SPEC-FR-04.4](../functional/FR-04/SPEC-FR-04.4.md) |
| SPEC-FR-04.1-v2 | Full reasoning loop | FR-04.1 | ✅ IMPLEMENTED | (agent_service.go) |
| SPEC-FR-10.2-M2 | Integration tests (Redis, Qdrant) | FR-10.2 | ⬜ PENDING | — |
| SPEC-FR-16 | S3 object storage (payload offload) | FR-16.1–16.3 | ✅ IMPLEMENTED | [SPEC-FR-16](../functional/FR-16/SPEC-FR-16.md) |
| SPEC-NFR-CACHE | Redis infrastructure cache | NFR-CACHE | ✅ IMPLEMENTED | [SPEC-NFR-CACHE](../nonfunctional/SPEC-NFR-CACHE.md) |

## Atomic Tasks

| Task ID | Spec | Description | Status |
|---------|------|-------------|--------|
| TASK-M2-001 | SPEC-FR-04.2 | Redis outbound port interface | ✅ Done (ports.go) |
| TASK-M2-002 | SPEC-FR-04.2 | Redis adapter unit tests — 7 tests | ✅ Done |
| TASK-M2-003 | SPEC-FR-04.2 | Redis adapter implementation (sorted sets, TTL) | ✅ Done |
| TASK-M2-004 | SPEC-FR-04.2 | Redis adapter integration test (testcontainers) | ⬜ Pending |
| TASK-M2-005 | SPEC-FR-04.3 | Qdrant outbound port interface | ✅ Done (ports.go) |
| TASK-M2-006 | SPEC-FR-04.3 | Qdrant adapter unit tests — 7 tests | ✅ Done |
| TASK-M2-007 | SPEC-FR-04.3 | Qdrant adapter implementation (upsert, search, filter) | ✅ Done |
| TASK-M2-008 | SPEC-FR-04.3 | Qdrant adapter integration test (testcontainers) | ⬜ Pending |
| TASK-M2-009 | SPEC-FR-04.4 | MCP outbound port interface | ✅ Done (ports.go) |
| TASK-M2-010 | SPEC-FR-04.4 | MCP adapter unit tests — 6 tests | ✅ Done |
| TASK-M2-011 | SPEC-FR-04.4 | MCP adapter implementation (CallTool, ListTools) | ✅ Done |
| TASK-M2-012 | SPEC-FR-04.1-v2 | Full reasoning loop tests | ✅ Done (agent service) |
| TASK-M2-013 | SPEC-FR-04.1-v2 | Full reasoning loop implementation | ✅ Done |
| TASK-M2-014 | SPEC-FR-16 | BlobStore outbound port interface | ✅ Done |
| TASK-M2-015 | SPEC-FR-16 | S3/MinIO adapter unit tests — 6 tests | ✅ Done |
| TASK-M2-016 | SPEC-FR-16 | S3/MinIO adapter implementation (Put/Get/Delete/Exists) | ✅ Done |
| TASK-M2-017 | SPEC-FR-16 | S3 adapter integration test (testcontainers MinIO) | ⬜ Pending |
| TASK-M2-018 | SPEC-FR-16 | Payload offload logic tests | ✅ Done |
| TASK-M2-019 | SPEC-FR-16 | Payload offload implementation | ✅ Done |
| TASK-M2-020 | SPEC-NFR-CACHE | Cache outbound port interface | ✅ Done |
| TASK-M2-021 | SPEC-NFR-CACHE | Redis cache adapter unit tests — 6 tests | ✅ Done |
| TASK-M2-022 | SPEC-NFR-CACHE | Redis cache adapter implementation (JSON, namespaced) | ✅ Done |
| TASK-M2-023 | SPEC-NFR-CACHE | Cache integration test (testcontainers) | ⬜ Pending |
| TASK-M2-024 | ALL | Refactor pass: adapter abstractions | ⬜ Pending |

