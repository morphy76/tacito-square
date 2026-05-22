# SPEC-NFR-HEXAGONAL: Hexagonal Architecture with Domain-Driven Design (DDD)

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-NFR-HEXAGONAL                 |
| Status        | ACCEPTED                           |
| Component     | agent, keeper                      |

## Specification

1. Each component (bounded context: agent, keeper) MUST follow hexagonal (ports & adapters) architecture integrated with Domain-Driven Design (DDD) principles.
2. The **domain** layer MUST NOT import any application, port, or adapter package. It encapsulates core business logic and invariant rules.
3. The **application** layer orchestrates domain objects to fulfill use cases. It MUST NOT import any adapter packages.
4. **Inbound ports** define use cases as Go interfaces consumed by driving adapters.
5. **Outbound ports** define dependency interfaces (e.g., Repositories) consumed by the application layer.
6. **Adapters** implement ports. Adapters may import domain and application types, but domain and application layers MUST NOT import adapters.
7. Package structure per component:
   - `domain/` — core business logic
     - `model/` — Aggregates, Entities, Value Objects, Domain Errors, State Machines
     - `event/` — Domain Events
     - `service/` — Domain Services (stateless logic spanning multiple aggregates)
   - `application/` — use case orchestration
     - `ports/inbound/` — use case interfaces
     - `ports/outbound/` — dependency interfaces (e.g., Repositories)
     - `service/` — use case implementations (depend on outbound ports)
   - `adapters/inbound/` — driving adapters (HTTP handlers, NATS subscribers)
   - `adapters/outbound/` — driven adapters (PostgreSQL repositories, OpenAI, Redis, Qdrant, NATS, MCP clients)

## Acceptance Criteria

1. `go vet ./internal/agent/domain/...` compiles with zero imports from `application/` or `adapters/` packages.
2. `go vet ./internal/keeper/domain/...` compiles with zero imports from `application/` or `adapters/` packages.
3. `go vet ./internal/*/application/...` compiles with zero imports from `adapters/` packages.
4. All application service constructors accept outbound port interfaces, not concrete adapters.
5. Persistence and infrastructure MUST be abstracted via interfaces defined in `application/ports/outbound/` and implemented in `adapters/outbound/`.
