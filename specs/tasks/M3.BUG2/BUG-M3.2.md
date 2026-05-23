# BUG-M3.2: Silent Route Registration Failure due to PostgreSQL Coupling in Keeper Bootstrap

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M3.2                                                           |
| Status        | OPEN                                                               |
| Severity      | HIGH                                                               |
| Milestone     | M3 — Keeper Core                                                   |
| Affects       | TASK-M3.1.2, TASK-M3.2.2, TASK-M3.3.2, TASK-M3.4.2                 |
| Violates      | SPEC-NFR-HEXAGONAL §6, SPEC-NFR-HTTP §4, SPEC-NFR-HEALTH §1, §3    |
| Discovered    | Post-implementation review against specs/nonfunctional requirements |

## Problem Statement

In the Keeper component, the initialization and registration of all REST API endpoints under `/api/v1` are tightly coupled to the presence of a non-nil `*pgxpool.Pool` database connection pool. 

Inside `internal/keeper/bootstrap.go`, the route registration logic for all domain entities (**LLM Bindings**, **MCP Servers**, **Skill Collections**, **Prompt Collections**, **Agents**, and **Communities**) is wrapped entirely in an `if pool != nil` guard:

```go
	if pool != nil {
		repo := postgres.NewLLMBindingRepository(pool)
		handler := httpAdapter.NewLLMBindingHandler(repo)
		// ...
		v1 := r.Group("/api/v1")
		v1.Use(httpAdapter.TenantResolutionMiddleware(httpAdapter.NewHeaderTenantResolver()))
		{
			v1.POST("/llm-bindings", handler.Create)
			// ...
		}
	}
```

If the PostgreSQL database is unavailable or unconfigured (such as during environment dry-runs, tests, or initial startup phases where connection fails or is delayed), the database pool `pool` is either `nil` or fails to initialize. In these cases:
1. **Silent Route Registration Failure**: The Gin HTTP engine starts successfully and serves auxiliary endpoints like `/healthz`, `/readyz`, `/metrics`, and `/openapi.json`, but completely fails to register any of the core REST API endpoints under `/api/v1`.
2. **Incorrect Error Codes (404 instead of 503/500)**: Any client or operator attempt to access entity endpoints (e.g. `GET /api/v1/agents`) returns a `404 Not Found` response instead of a proper `503 Service Unavailable` or `500 Internal Server Error` representing a database down state.
3. **Violation of Architectural Specifications**: 
   - **SPEC-NFR-HEXAGONAL**: Exposing API routes dynamically based on external database connectivity violates the predictability of the service API. Additionally, the bootstrapping code is tightly coupled to concrete postgres repository instantiations rather than abstracting repository interfaces or maintaining decoupled dependency injection.
   - **SPEC-NFR-HTTP**: Dynamic route registration conflicts with centralized route registration principles.
   - **SPEC-NFR-HEALTH**: An unavailable database must be reflected in the `/readyz` probe reporting 503, but the application's actual routing layout must remain deterministic and stable.

## Affected Aggregates and Files

| File / Component | Location | Issue |
|------------------|----------|-------|
| `bootstrap.go` | `internal/keeper/bootstrap.go` | Route registration for `/api/v1` gated behind `if pool != nil`. Concrete Postgres repositories instantiated directly inside `NewServer`. |
| `main.go` | `cmd/keeper/main.go` | Hard exit/panic on PostgreSQL connection errors rather than letting the server start up and handle/report dependency health via `/readyz`. |

## Impact

1. **Undeterministic API routing**: The routing table changes depending on database availability at startup. An unavailable database causes endpoints to vanish entirely (404), misleading downstream API clients, reverse proxies, and operators.
2. **Broken Health Probe Contract**: Under `SPEC-NFR-HEALTH`, the application is supposed to start up, register its routes, and report the database status dynamically via `/readyz`. With the current implementation, if the DB fails to connect at startup, the app might crash (due to `logger.Fatal().Err(err).Msg("failed to connect to postgres")` in `cmd/keeper/main.go`) or start with missing routes.
3. **Impaired Testability**: Integration and unit tests that do not provision a full PostgreSQL instance cannot test the routing structure, middleware bindings, or request validation of the HTTP layer.

## Expected Behaviour (per SPEC-NFR-HEXAGONAL, SPEC-NFR-HTTP & SPEC-NFR-HEALTH)

1. The HTTP routing table for all `/api/v1` endpoints MUST be registered unconditionally during `NewServer` bootstrapping, regardless of the database connection state.
2. If the PostgreSQL pool is uninitialized or disconnected when an API request is received, the system MUST return a structured JSON error with `503 Service Unavailable` or `500 Internal Server Error`, rather than a `404 Not Found`.
3. The bootstrap function (`NewServer`) should decouple from concrete repository instantiation, allowing either mock/test repositories or safely injecting repository dependencies (satisfying hexagonal architecture).
4. The database connection check must be evaluated dynamically in the `/readyz` health check (per `SPEC-NFR-HEALTH`), allowing the service to run and report status rather than crashing immediately at startup or starting in a partially-registered state.

## Acceptance Criteria

1. All `/api/v1` routes are registered unconditionally when calling `NewServer`, even with a `nil` or unconfigured PostgreSQL pool.
2. Accessing any `/api/v1` endpoint when the database is unavailable yields a standard JSON error response: `{"error": "Database service unavailable"}` with a `503 Service Unavailable` or `500 Internal Server Error` HTTP status code.
3. Decouple `bootstrap.go` from concrete `*pgxpool.Pool` repository constructors, using abstract repository ports or registering handlers with proper dependency checks.
4. Unit and integration tests can verify routing and middleware behavior (e.g. `TestNewServer_ReturnsGinEngine`) without needing a live PostgreSQL database connection pool.
5. `/readyz` properly returns `503` when the PostgreSQL database checker fails, but the API endpoints still exist in the router.
