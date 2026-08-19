---
trigger: glob
globs: ["**/*.go", "**/*.ts", "**/*.tsx"]
description: Hexagonal (Ports & Adapters) design pattern, DDD layering boundaries, and Go reactive concurrency paradigms.
---

# Code Architecture Guidelines

This rule enforces structural alignment with Hexagonal (Ports & Adapters) architecture, Domain-Driven Design (DDD) boundaries, and Go reactive concurrency standards as specified in `SPEC-NFR-HEXAGONAL` and `SPEC-NFR-REACTIVE`.

## 1. Hexagonal Architecture with DDD (SPEC-NFR-HEXAGONAL)

All core components (specifically `agent` and `keeper` bounded contexts) must adhere to a strict clean-architecture separation:

### A. Layering and Import Constraints
- **Domain Layer (`domain/`):** Contains core business aggregates, entities, value objects, domain errors, state machines, events, and stateless domain services.
  - **CRITICAL CONSTRAINT:** The domain layer MUST NOT import any packages from the `application` layer or the `adapters` layer. It must remain pure business logic.
- **Application Layer (`application/`):** Orchestrates domain objects to execute specific use cases.
  - **CRITICAL CONSTRAINT:** The application layer MUST NOT import any packages from the `adapters` layer. It has zero knowledge of database technologies, NATS messaging, or external HTTP clients.
- **Adapters Layer (`adapters/`):** Implements infrastructure bindings.
  - Adapters may import both domain and application layers, but domain and application layers must never import concrete adapters.

### B. Ports and Package Structure
Structure every component context under `internal/<component>/` using the following exact directory hierarchy:
```text
internal/<component>/
├── domain/                  # Core Business Domain Layer (pure)
│   ├── model/               # Aggregates, Entities, Value Objects, Domain Errors, State Machines
│   ├── event/               # Domain Events
│   └── service/             # Domain Services (cross-aggregate logic)
├── application/             # Use Case Orchestration Layer
│   ├── ports/
│   │   ├── inbound/         # driving interfaces (consumed by HTTP handlers, NATS subscribers)
│   │   └── outbound/        # driven interfaces (repositories, Redis caches, pub-sub adapters)
│   └── service/             # use case implementations (depends ONLY on outbound port interfaces)
└── adapters/                # Infrastructure & I/O Adapters Layer
    ├── inbound/             # Driving Adapters (Gin HTTP handlers, NATS subscribers)
    └── outbound/            # Driven Adapters (pgx databases, OpenAI, Redis, Qdrant, NATS clients)
```

### C. Dependency Injection
- Always define dependencies as interfaces inside `application/ports/outbound/`.
- Concrete adapters in `adapters/outbound/` must implement these interfaces.
- Application service constructors must accept outbound port *interfaces*, never concrete adapters.

## 2. Reactive Programming (SPEC-NFR-REACTIVE)

Prioritize highly concurrent, reactive workflows over traditional sequential/imperative code:

- **Go Concurrency Primitives:** Leverage Go Goroutines, Channels, and `select` blocks to build responsive, non-blocking pipelines.
- **Asynchronous Execution:** Heavy computation, external API network calls, and complex I/O pipelines must not block execution synchronously. Wrap them in goroutines and coordinate via channels.
- **Event-Driven Coupling:** Favor decoupling components by emitting events to asynchronous topics (e.g., NATS) or channels.
- **Context Management:** Standard Go `context.Context` must be actively propagated across all concurrent and asynchronous goroutine boundaries. Always manage context cancellations to prevent goroutine leaks.

---

## Developer Checklists & Verifications

- [ ] Does my domain package contain imports from application or adapters? (Should be empty).
- [ ] Does my application service depend on any concrete database or network clients? (Must only depend on interfaces in `application/ports/outbound/`).
- [ ] Is my dependency injected via an interface in the constructor?
- [ ] Am I avoiding long-running synchronous blocking loops?
- [ ] Are all of my asynchronous goroutines managed by a parent `context.Context` to prevent resource leaks?
