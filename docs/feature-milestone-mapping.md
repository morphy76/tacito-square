# Tacito Square — Feature & Milestone Mapping

## Current Status

| Milestone | Status | Tests | Description |
|-----------|--------|-------|-------------|
| **M1: Foundation** | ✅ Complete | 64 | Walking skeleton — agent + keeper + Gin + zerolog + OTel + health + Helm |
| M2: Memory & Tools | ⬜ Planned | — | Redis, Qdrant, MCP adapters |
| M3: Prompt & Skills | ⬜ Planned | — | Versioned prompts, skill CRUD |
| M4: Communities & HITL | ⬜ Planned | — | Threads, Hub-Spoke, callback hooks |
| M5: K8s Operator | ⬜ Planned | — | CRDs, reconcilers, webhooks |
| M6: UIs & BFF | ⬜ Planned | — | React 19, OIDC login |
| M7: A2A Gateway | ⬜ Planned | — | External source registry, federation |
| M8: Hardening | ⬜ Planned | — | Benchmarks, E2E, CI/CD |

## Feature → Milestone Mapping

| FR | Feature | Milestone | Status |
|----|---------|-----------|--------|
| FR-01.1 | Spawn agent from prompt + skills | M1, M3 | 🟡 M1 domain done |
| FR-01.2 | Agent state transitions | M1 | ✅ Done |
| FR-01.3 | Config snapshot at spawn | M1 | ✅ Done |
| FR-01.4 | Heartbeat processing | M1 | ⬜ |
| FR-01.5 | Terminate agents | M1 | ✅ Done |
| FR-01.6 | Audit log per transition | M4 | ⬜ |
| FR-02.1 | Prompt CRUD | M3 | ⬜ |
| FR-02.2 | Prompt versioning | M3 | ⬜ |
| FR-02.3 | Prompt import/export | M3 | ⬜ |
| FR-03.1 | Skill CRUD | M3 | ⬜ |
| FR-03.2 | MCP tool attach/detach | M3 | ⬜ |
| FR-03.3 | Skill assignment at spawn | M3 | ⬜ |
| FR-04.1 | LLM reasoning loop | M1, M2 | 🟡 Brain adapter done |
| FR-04.2 | Short-term memory (Redis) | M2 | ⬜ |
| FR-04.3 | Long-term memory (Qdrant) | M2 | ⬜ |
| FR-04.4 | Tool invocation (MCP) | M2 | ⬜ |
| FR-04.5 | Specialist agent spawn | M4 | ⬜ |
| FR-04.6 | Conversation handoff | M4 | ⬜ |
| FR-04.7 | HITL `input-required` yield | M4 | ⬜ |
| FR-05.1 | Community management | M4 | ⬜ |
| FR-05.2 | Hub-Spoke topology | M4 | ⬜ |
| FR-05.3 | NATS subject namespacing | M4 | ⬜ |
| FR-05.4 | K8s NetworkPolicies | M4 | ⬜ |
| FR-05.5 | Multi-thread engagements | M4 | ⬜ |
| FR-05.6 | Thread CRUD | M4 | ⬜ |
| FR-06.1 | A2A Agent Cards | M4 | ⬜ |
| FR-06.2 | NATS internal messaging | M1 | ⬜ |
| FR-06.3 | A2A HTTP gateway | M7 | ⬜ |
| FR-06.4 | Hub routing | M4 | ⬜ |
| FR-06.5 | External source registry | M7 | ⬜ |
| FR-06.6 | External agent messaging | M7 | ⬜ |
| FR-06.7 | External source health | M7 | ⬜ |
| FR-07.1 | Agent CRD | M5 | ⬜ |
| FR-07.2 | AgentCommunity CRD | M5 | ⬜ |
| FR-07.3 | Validating webhooks | M5 | ⬜ |
| FR-07.4 | Mutating webhooks | M5 | ⬜ |
| FR-08.1 | Keeper REST API | M1 | ✅ Done |
| FR-08.2 | User REST API | M6 | ⬜ |
| FR-08.3 | Keeper UI | M6 | ⬜ |
| FR-08.4 | User UI | M6 | ⬜ |
| FR-08.5 | OIDC/JWT auth | M1, M6 | 🟡 Stub done |
| FR-08.6 | BFF layer | M6 | ⬜ |
| FR-09.1 | OpenTelemetry tracing | M1 | ✅ Done |
| FR-09.2 | Structured logging (zerolog) | M1 | ✅ Done |
| FR-09.3 | Prometheus metrics | M8 | ⬜ |
| FR-09.4 | Audit log queries | M4 | ⬜ |
| FR-10.1 | Unit tests (TDD) | M1+ | ✅ Active |
| FR-10.2 | Integration tests | M1 | 🟡 In progress |
| FR-10.3 | Operator tests | M5 | ⬜ |
| FR-10.4 | E2E tests | M8 | ⬜ |
| FR-10.5 | Benchmark tests | M8 | ⬜ |
| FR-10.6 | Concurrency tests | M8 | ⬜ |
| FR-10.7 | Makefile targets | M1 | ✅ Done |
| FR-11.1 | HITL Agent Card flag | M4 | ⬜ |
| FR-11.2 | HITL yield in reasoning | M4 | ⬜ |
| FR-11.3 | HITL callback persistence | M4 | ⬜ |
| FR-11.4 | HITL human response | M4 | ⬜ |
| FR-11.5 | HITL TTL/escalation | M4 | ⬜ |
| FR-11.6 | HITL audit events | M4 | ⬜ |
| FR-12.1 | Bearer JWT auth | M1 | ✅ Done |
| FR-12.2 | API-first design | M1 | ✅ Done |
| FR-12.3 | Independent versioning | M1 | ✅ Done |
| FR-12.4 | OpenAPI contracts | M1 | ⬜ |
| FR-12.5 | Contract tests | M8 | ⬜ |
| FR-12.6 | Helm sub-charts | M1 | ✅ Done |
| FR-13.1 | RBAC role model | M6 | 🟡 Keycloak realm done |
| FR-13.2 | Gin RBAC middleware | M6 | ⬜ |
| FR-13.3 | Principal logging on mutations | M6 | ⬜ |
| FR-13.4 | Audit trail with actor identity | M4, M6 | ⬜ |
| FR-13.5 | Keycloak realm via Helm | M1 | ✅ Done |
| FR-13.6 | Role-based route protection | M6 | ⬜ |
| FR-13.7 | Service-to-service auth | M6 | ⬜ |
| NFR-LOG | zerolog + trace_id + claims | M1 | ✅ Done |
| NFR-HTTP | Gin HTTP framework | M1 | ✅ Done |
