# Milestone M1: Foundation (Walking Skeleton)

| Field      | Value |
|------------|-------|
| Status     | ✅ COMPLETE |
| Tests      | 64 |
| Packages   | 11 |

## Goal

A single agent can be spawned by the Keeper, receive a message via NATS, reason with an LLM, and respond. Deployable on Kind via `helm install`.

## Deliverable

`helm install tacito-square` on Kind → spawn agent via curl → agent responds via NATS.

## Specs Covered

| Spec ID | Title | Status |
|---------|-------|--------|
| SPEC-FR-01.1 | Spawn agent from prompt + skills | ✅ Domain done |
| SPEC-FR-01.2 | Agent state transitions | ✅ Verified |
| SPEC-FR-01.3 | Config snapshot at spawn | ✅ Verified |
| SPEC-FR-01.5 | Terminate agents | ✅ Verified |
| SPEC-FR-04.1 | LLM reasoning (brain adapter) | ✅ Verified |
| SPEC-FR-08.1 | Keeper REST API | ✅ Verified |
| SPEC-FR-08.5 | OIDC/JWT auth stub | ✅ Verified |
| SPEC-FR-09.1 | OpenTelemetry tracing | ✅ Verified |
| SPEC-FR-09.2 | Structured logging (zerolog) | ✅ Verified |
| SPEC-FR-10.1 | Unit tests (TDD) | ✅ Active |
| SPEC-FR-10.7 | Makefile targets | ✅ Verified |
| SPEC-FR-12.1 | Bearer JWT auth | ✅ Verified |
| SPEC-FR-12.2 | API-first design | ✅ Verified |
| SPEC-FR-12.3 | Independent versioning | ✅ Verified |
| SPEC-FR-12.6 | Helm sub-charts | ✅ Verified |
| SPEC-FR-13.5 | Keycloak realm via Helm | ✅ Verified |
| SPEC-NFR-LOG | zerolog + trace_id + claims | ✅ Verified |
| SPEC-NFR-HTTP | Gin HTTP framework | ✅ Verified |
| SPEC-NFR-HEALTH | Dependency-aware health probes | ✅ Verified |

## Implementation Summary

### Packages Delivered

| Package | Tests | Description |
|---------|-------|-------------|
| `internal/agent/domain` | 11 | Agent, Message, MemoryEntry, ToolDescriptor |
| `internal/agent/service` | 3 | AgentService (HandleMessage, GetCard) |
| `internal/agent/adapters/outbound/openai` | 3 | OpenAI brain adapter |
| `internal/keeper/domain` | 11 | AgentInstance state machine, SpawnRequest |
| `internal/keeper/service` | 5 | KeeperService (Spawn, List, Get, Terminate) |
| `internal/keeper/adapters/inbound/httphandler` | 5 | Gin REST API |
| `internal/shared/config` | 3 | Viper config loader |
| `internal/shared/observability` | 6 | zerolog logger + OTel tracing |
| `internal/shared/errors` | 4 | Domain error types |
| `internal/shared/auth` | 7 | Bearer/JWT context handling |
| `internal/shared/health` | 7 | Health probes |

### Infrastructure

- Helm umbrella chart with NATS, Redis, PostgreSQL, Qdrant, OTel Collector, Keycloak
- Kind cluster config, Dockerfiles (multi-stage, distroless), Makefile (20+ targets)
