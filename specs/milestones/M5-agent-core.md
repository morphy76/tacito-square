# Milestone M5: Agent Core

| Field      | Value |
|------------|-------|
| Status     | ⏳ IN_PROGRESS |

## Goal

Agent loads configuration from CRD spec, reasons via LLM, uses short-term and long-term memory, invokes tools via MCP, and stores large payloads in S3-compatible object storage.

## Deliverable

Spawned agent → loads config → receives message via NATS → reasons with LLM → stores/retrieves memory → invokes MCP tools → responds.

## Specs Required

| Spec ID | Title | Component | Depends On |
|---------|-------|-----------|------------|
| SPEC-FR-M5.1 | Agent Configuration from CRD Spec | agent | SPEC-FR-M4.1 |
| SPEC-FR-M5.2 | LLM Reasoning (Brain Adapter) | agent | SPEC-FR-M5.1 |
| SPEC-FR-M5.3 | Short-Term Memory (Redis) | agent | SPEC-FR-M5.1 |
| SPEC-FR-M5.4 | Long-Term Memory (Qdrant) | agent | SPEC-FR-M5.1 |
| SPEC-FR-M5.5 | Tool Invocation (MCP) | agent | SPEC-FR-M5.2 |
| SPEC-FR-M5.6 | Object Storage (S3/MinIO) | agent | SPEC-FR-M5.1 |
