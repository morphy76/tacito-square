# SPEC-ARCH-001: Hexagonal Architecture

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-ARCH-001                      |
| Status        | VERIFIED                           |
| Component     | agent, keeper                      |

## Specification

1. Each component (agent, keeper) MUST follow hexagonal (ports & adapters) architecture.
2. The **domain** layer MUST NOT import any adapter or infrastructure package.
3. **Inbound ports** define use cases as Go interfaces consumed by driving adapters.
4. **Outbound ports** define dependency interfaces consumed by the domain/service layer.
5. **Adapters** implement ports. Adapters may import domain types but domain MUST NOT import adapters.
6. Package structure per component:
   - `domain/` — entities, value objects, domain errors, state machines
   - `ports/inbound/` — use case interfaces
   - `ports/outbound/` — dependency interfaces
   - `service/` — use case implementations (depend on outbound ports)
   - `adapters/inbound/` — HTTP handlers, NATS subscribers
   - `adapters/outbound/` — OpenAI, Redis, Qdrant, PostgreSQL, NATS, MCP clients

## Acceptance Criteria

1. `go vet ./internal/agent/domain/...` compiles with zero imports from `adapters/`
2. `go vet ./internal/keeper/domain/...` compiles with zero imports from `adapters/`
3. All service constructors accept port interfaces, not concrete adapters
