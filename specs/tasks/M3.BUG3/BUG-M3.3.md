# BUG-M3.3: Hexagonal Architecture Violations (Missing Application Service Layer & Flat Bounded Contexts)

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M3.3                                                           |
| Status        | CLOSED                                                             |
| Severity      | HIGH                                                               |
| Milestone     | M3 — Keeper Core                                                   |
| Affects       | internal/keeper/application, internal/keeper/adapters, internal/keeper/domain |
| Violates      | SPEC-NFR-HEXAGONAL §1, §2, §3, §4, §5, §6, §7                      |
| Discovered    | M3 candidate post-implementation NFR review                         |

## Problem Statement

The current Milestone M3 candidate implementation for `keeper` does not conform to the strict Hexagonal (Ports & Adapters) Architecture and Domain-Driven Design (DDD) requirements defined in `SPEC-NFR-HEXAGONAL`.

Specifically, the following structural violations exist:
1. **Missing Application Service Layer**: There are no application services or use case implementations under `internal/keeper/application/`. Instead, the driving HTTP adapters (e.g., `AgentHandler` in `internal/keeper/adapters/http/agent_handlers.go`) directly consume outbound repository ports (e.g., `outbound.AgentRepository`) and perform business orchestration, parameter validation, and mapping logic.
2. **Missing Inbound Ports**: No use case interfaces are defined under `internal/keeper/application/ports/inbound/`.
3. **Flat Domain Layout**: The domain aggregates (e.g., `llm_binding.go`, `skill.go`, `agent.go`) are flatly located inside `internal/keeper/domain/` rather than structured under `domain/model/`, `domain/event/`, and `domain/service/` folders as requested.
4. **Adapter Packaging Mismatch**: Concrete driving and driven adapters are organized as flat `adapters/http/` and `adapters/postgres/` directories rather than being partitioned under the standard `adapters/inbound/` and `adapters/outbound/` layout.

## Affected Aggregates and Files

| Component / Layer | Current File / Directory | Issue |
|-------------------|--------------------------|-------|
| `application` | `internal/keeper/application/` | Lacks `service/` and `ports/inbound/`. Completely bypassed. |
| `adapters/inbound` | `internal/keeper/adapters/http/` | Direct coupling with outbound repositories; wrong package location. |
| `adapters/outbound` | `internal/keeper/adapters/postgres/` | Wrong package location. |
| `domain` | `internal/keeper/domain/` | Flat structure; lacks separation of model aggregates and domain services. |

## Impact

1. **Poor Maintainability**: Mixing HTTP request parsing and domain workflow orchestration inside controllers leads to large, monolithic handlers that are difficult to unit-test in isolation from HTTP routers.
2. **Lack of DDD Invariants Enforcement**: The lack of a dedicated application service layer makes it hard to enforce business invariants that cross multiple aggregates without cluttering the HTTP layer.
3. **Violation of Architectural Contract**: Direct imports of outbound ports in the driving adapters bypasses standard hexagonal decoupling, conflicting with SPEC-NFR-HEXAGONAL guidelines.

## Expected Behaviour

1. Driving adapters (HTTP controllers) must ONLY import and delegate workflows to inbound port interfaces defined in `internal/keeper/application/ports/inbound/`.
2. Inbound ports must be implemented by concrete application services located in `internal/keeper/application/service/`.
3. Application services must orchestrate domain aggregates and call outbound ports (`internal/keeper/application/ports/outbound/`), with zero imports of adapters.
4. The codebase must follow the standard hexagonal and DDD folder layout:
   - `internal/keeper/domain/model/` for models.
   - `internal/keeper/adapters/inbound/http/` for driving controllers.
   - `internal/keeper/adapters/outbound/postgres/` for driven DB pools.

## Acceptance Criteria

1. Unit tests pass with `go vet ./internal/keeper/domain/...` compilation yielding zero imports from `application/` or `adapters/`.
2. All business logic and domain coordination are extracted from Gin controllers into Application Services.
3. Gin HTTP handlers only handle JSON marshalling, path binding, dynamic header resolution, and delegate orchestration to inbound services.
4. Package directories conform strictly to the standard directories outlined in SPEC-NFR-HEXAGONAL.
