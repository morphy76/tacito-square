# Codebase Hotspots Registry

> Track and prioritize Go and TypeScript files with high complexity or length (>500 LOC) to refactor them incrementally.

## Active Hotspots Queue

| Priority | File Path | Lang | LOC | Complexity Issues | Status |
| :---: | :--- | :---: | :---: | :--- | :---: |
| 1 | [internal/agent/application/service/cognitive_engine.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/application/service/cognitive_engine.go) | Go | 858 | God class: orchestrates reasoning loop, handles built-in tools (LTM, S3, skills), OTEL metrics/tracing, and NATS event emission. High cognitive complexity. | RESOLVED |
| 2 | [internal/keeper/adapters/outbound/postgres/agent_repository.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/outbound/postgres/agent_repository.go) | Go | 686 | Architecture violation: performs community topology & role validation inside SQL repository transactions instead of domain/application service. | RESOLVED |
| 3 | [internal/keeper/adapters/inbound/http/skill_handlers.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/inbound/http/skill_handlers.go) | Go | 670 | Heavy handler logic, manual bindings, and multi-layered route configurations. | QUEUED |
| 4 | [internal/operator/application/service/reconcile_service.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/operator/application/service/reconcile_service.go) | Go | 620 | Complex controller-like reconciliation state machine with deep nesting. | QUEUED |
| 5 | [internal/agent/adapters/outbound/redis/memory_adapter.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/adapters/outbound/redis/memory_adapter.go) | Go | 609 | Complex Redis pipeline transactions, serialization patterns, and manual trace correlation. | QUEUED |
| 6 | [internal/keeper/adapters/inbound/http/prompt_handlers.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/inbound/http/prompt_handlers.go) | Go | 573 | High density of redundant validation, formatting, and pagination logic in route handlers. | QUEUED |
| 7 | [internal/agent/adapters/outbound/ollama/ollama_adapter.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/adapters/outbound/ollama/ollama_adapter.go) | Go | 518 | Complex chat template builder, HTTP retry logic, and stream-buffered parsing. | QUEUED |
| 8 | [internal/agent/adapters/outbound/openai/openai_adapter.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/adapters/outbound/openai/openai_adapter.go) | Go | 504 | Heavy client mapping, context timeouts, and custom OTel wrapper. | QUEUED |
| 9 | [internal/agent/application/service/orchestrator.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/application/service/orchestrator.go) | Go | 503 | Background worker routing, dynamic thread pools, and complex event subscriptions. | QUEUED |

## Refactoring History

| Resolved Date | File Path | Lang | Original LOC | New LOC | Improvements Achieved | Task Link |
| :--- | :--- | :---: | :---: | :---: | :--- | :---: |
| 2026-06-08 | [internal/agent/application/service/cognitive_engine.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/application/service/cognitive_engine.go) | Go | 858 | 581 | Extracted context helpers, structs, and built-in tools into modular service files. | [TASK-REFACTOR-cognitive_engine](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/tasks/refactor/TASK-REFACTOR-cognitive_engine.md) |
| 2026-06-14 | [internal/keeper/adapters/outbound/postgres/agent_repository.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/outbound/postgres/agent_repository.go) | Go | 686 | 686 | Extracted community topology, role validation, and status check logic to AgentService.Assign. | [TASK-REFACTOR-agent_repository](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/tasks/refactor/TASK-REFACTOR-agent_repository.md) |
