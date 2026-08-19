---
trigger: glob
globs: ["**/*.go", "**/*.ts", "**/*.tsx"]
description: Cloud-First Development patterns, multitenancy design, API-First principles, circuit breakers, and live OpenAPI specs.
---

# Cloud-First Development & Multitenancy Guidelines

This rule enforces patterns for cloud-first design, secure multi-tenant isolation, and automated API documentation as specified in `SPEC-NFR-CLOUD`, `SPEC-NFR-MULTITENANCY`, `SPEC-NFR-OPENAPI`, and Principles P3 and P4 of the Project Constitution.

## 1. Cloud-First Patterns (SPEC-NFR-CLOUD)

When implementing or modifying code that interacts with external resources or network APIs:

- **Statelessness:** Ensure all application instances remain stateless between requests. Never store session, transactional, or persistent state in local memory or filesystem. Use PostgreSQL or Redis.
- **Circuit Breakers:** Wrap every outbound external service call (e.g., LLM providers, external third-party APIs, and external databases) in tested circuit breakers.
- **Retries and Backoff:** Implement automatic retries for transient failures using an exponential backoff strategy with random jitter to prevent thundering herd conditions.
- **Timeouts and Deadlines:** Enforce explicit, configurable timeouts on every outbound network request and long-running operation. Always propagate timeouts using Go's `context.Context` across package and RPC boundaries.
- **Rate Limiting:** Protect all inbound APIs (especially public/authenticated endpoints or resource-heavy routes) using rate-limiting algorithms (e.g., token bucket).
- **Bulkheads:** Set explicit resource boundaries, connection pool limits, and isolation segments so that resource exhaustion in one service doesn't cascade.
- **Graceful Degradation:** Design systems to degrade gracefully. If a non-critical dependency fails, return partial cached data or a simplified response instead of throwing a generic error.

## 2. Multitenancy Architecture (SPEC-NFR-MULTITENANCY)

Understand the distinction between the two tenant deployment units:

### A. Dedicated Single-Tenant Units (e.g., individual Agent, Agent Community pods)
- **Tenant Resolution:** Read the tenant ID strictly from environment configuration (e.g., `TENANT_ID`).
- **Data Isolation:** Qualify/scope all outbound port calls (database, cache, vector search) using this statically injected tenant identity.
- **Access Control:** Reject any request whose payload, headers, or metadata do not match the configured `TENANT_ID`.

### B. Shared Multi-Tenant Units (e.g., Keeper, BFF, API Gateways)
- **Tenant Resolution:** Dynamically extract the tenant ID at runtime:
  - **REST (Authenticated):** Extract from the JWT token claims parsed by Zitadel/OIDC middleware.
  - **REST (Unauthenticated):** Extract from HTTP headers (e.g., `X-Tenant-ID`).
  - **Messaging:** Extract from NATS message headers or NATS subject context.
- **Propagation:** Always propagate the resolved tenant ID across package boundaries via Go's `context.Context`.
- **Access Control:** Enforce tenant isolation at the database/storage query layers (e.g., `WHERE tenant_id = ?` in pgx queries). Reject requests with a clear 401/403 or validation error if the tenant ID is missing or invalid.

## 3. API-First, Contract-Based & OpenAPI Design (P3/P4 & SPEC-NFR-OPENAPI)

Enforce decoupled, interface-first cloud patterns:

- **API-First Design (Principle P3):** All business capabilities MUST be accessible and fully functional via authenticated REST APIs. User interfaces are optional, secondary consumers of these stable underlying endpoints.
- **Contract-Based Isolation (Principle P4):** Distinct components MUST interact strictly through versioned contracts (e.g., OpenAPI descriptions or NATS message topic payloads), enabling independent versioning and deployment lifecycles.
- **OpenAPI 3.x Specifications (SPEC-NFR-OPENAPI):**
  - **Spec Path:** Every HTTP-exposed service MUST serve its valid OpenAPI 3.x specification at `GET /openapi.json`.
  - **Swagger UI:** Enable Swagger UI under `GET /swagger/` in development mode only (disable in production).
  - **Boundary Tagging:**
    - Every operation MUST carry exactly one tag in the format `domain/subdomain` (e.g., `infrastructure/llm-bindings`, `agent-config/skills`).
    - The top-level `tags` list must declare all these tags with descriptions naming their bounded context.
  - **Fidelity:** Maintain code annotations or build-embedded files to keep the live spec in sync with the contract in `api/openapi/`.

---

## Developer Checklists & Verifications

- [ ] Does my outbound call wrap errors and run within a `context.Context` deadline?
- [ ] Are external client libraries configured with proper circuit breaker and retry mechanisms?
- [ ] Do dedicated units read `TENANT_ID` from the environment?
- [ ] Do shared service endpoints validate and propagate tenant context?
- [ ] Is my new capability exposed fully via a REST API before any UI integration?
- [ ] Does my new HTTP API endpoint serve valid OpenAPI docs with a `domain/subdomain` tag?
