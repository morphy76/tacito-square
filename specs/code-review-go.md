# Tacito Square — Go Codebase Code Review

> Date: 2 June 2026
> Scope: All Go files under `cmd/`, `internal/`, `pkg/kubernetes/apis/`, `test/contract/` (≈168 files, excluding `vendor/` and generated `zz_*`).
> Method: Static review against Go idioms, hexagonal layering, OWASP Top 10, K8s controller patterns, OTel/observability best practices, multitenancy invariants, and concurrency safety.
> Severity legend: **CRITICAL** (data loss / security / outage) · **HIGH** (correctness / scalability) · **MED** (maintainability / minor functional risk) · **LOW** / **NIT** (polish).

---

## 1. Executive Summary

The codebase exhibits a **disciplined hexagonal architecture** consistently applied across the four Go components (Keeper, Operator, Agent, BFF), with shared cross-cutting code under [internal/shared](../internal/shared) and a clear `domain → application → adapters` layering. Naming is uniform, OTel tracing is wired pervasively, and the contract test in [test/contract/contract_test.go](../test/contract/contract_test.go) is genuinely impressive.

That said, the project is **not yet production-ready**. The main risk clusters are:

1. **Authentication / authorization** — there is no JWT signature validation anywhere. [internal/shared/auth/auth.go](../internal/shared/auth/auth.go) only parses the `Bearer` header; every service trusts whatever lands in `X-Tenant-ID` / `X-Subscription-ID`.
2. **BFF is a skeleton** — [internal/bff/bootstrap.go](../internal/bff/bootstrap.go) only exposes health probes; no proxy, no auth, no CORS, no metrics. The architectural diagram in [docs/architecture/overview.md](../docs/architecture/overview.md) does not match implementation.
3. **Multitenancy isolation in Qdrant** — a single shared collection for all tenants relying on payload filtering. One off-by-one in the filter builder is a cross-tenant data breach.
4. **Sensitive data in logs** — both [openai_adapter.go](../internal/agent/adapters/outbound/openai/openai_adapter.go) and [ollama_adapter.go](../internal/agent/adapters/outbound/ollama/ollama_adapter.go) log full request bodies (system + user prompts) at debug level.
5. **Concurrency hazards** — unsupervised goroutines in `agent_service.Assign` and `message_processor.triggerMemoryConsolidation`, no panic recovery in the NATS echo subscriber, and a sub-50 ms DB poll loop in `echo_service.waitForRunningAgents`.
6. **Operator reconciliation gaps** — no finalizers on `TacitoAgent`, full `Spec` overwrite on Deployment/Service updates (will fail on immutable `Selector`), no requeue/backoff on transient errors.

Risk level: **HIGH** for production · **MEDIUM** for staging · acceptable for development.

### 1.1 Top critical findings (fix before any production rollout)

| # | Finding | Location |
|---|---------|----------|
| C1 | No JWT signature/expiry/issuer validation; tenant headers are trusted blindly | [internal/shared/auth/auth.go](../internal/shared/auth/auth.go), [internal/keeper/adapters/inbound/http/middleware.go](../internal/keeper/adapters/inbound/http/middleware.go) |
| C2 | BFF has no gateway logic — no proxy, no auth, no CORS, no `/metrics` | [internal/bff/bootstrap.go](../internal/bff/bootstrap.go) |
| C3 | Full request body logged with prompts at debug level | [openai_adapter.go](../internal/agent/adapters/outbound/openai/openai_adapter.go), [ollama_adapter.go](../internal/agent/adapters/outbound/ollama/ollama_adapter.go) |
| C4 | Single shared Qdrant collection for all tenants | [ltm_adapter.go](../internal/agent/adapters/outbound/qdrant/ltm_adapter.go) |
| C5 | No finalizer on `TacitoAgent`; child resources orphaned on delete | [reconciler.go](../internal/operator/adapters/inbound/reconciler.go) |
| C6 | `existingDep.Spec = dep.Spec` clobbers immutable fields (`Selector`) on update | [reconcile_service.go](../internal/operator/application/service/reconcile_service.go) |
| C7 | Unsupervised goroutines in agent assign / memory consolidation; no `WaitGroup`, no shutdown wait | [agent_service.go](../internal/keeper/application/service/agent_service.go), [message_processor.go](../internal/agent/application/service/message_processor.go) |
| C8 | No `defer recover()` in NATS echo subscriber — one panic kills message handling | [echo_subscriber.go](../internal/agent/adapters/inbound/nats/echo_subscriber.go) |
| C9 | OTel tracer init leaks gRPC connections if metric exporter init fails | [internal/shared/observability/tracing.go](../internal/shared/observability/tracing.go) |
| C10 | N+1 in `agent_repository.List` (loads skills per agent) | [internal/keeper/adapters/outbound/postgres/agent_repository.go](../internal/keeper/adapters/outbound/postgres/agent_repository.go) |
| C11 | No optimistic locking on repository `Update` methods — concurrent writes silently overwrite | [internal/keeper/application/ports/outbound/repositories.go](../internal/keeper/application/ports/outbound/repositories.go) |
| C12 | `mockRedis` shipped in production binary | [internal/shared/adapters/outbound/cache/cache_adapter.go](../internal/shared/adapters/outbound/cache/cache_adapter.go) |

---

## 2. Cross-cutting strengths

- **Hexagonal layering is real, not aspirational** — `domain/model` types do not import gin, pgx, or k8s; ports are small focused interfaces; adapters compose them. Layering violations are rare and minor (see §6 for the few cases).
- **Bootstrap consistency** — all four `cmd/<svc>/main.go` entrypoints follow the same shape: viper config → zerolog → OTel tracer → `shutdown.Manager` (30 s) → service `NewServer()` → graceful shutdown.
- **OTel coverage** — server spans on every Gin handler, pgx tracer ([db_tracing.go](../internal/shared/observability/db_tracing.go)), W3C `traceparent` injection over NATS via `NATSHeaderCarrier` ([nats_tracing.go](../internal/shared/observability/nats_tracing.go)). The shared instrumented HTTP client is also used uniformly.
- **Contract test** — [test/contract/contract_test.go](../test/contract/contract_test.go) verifies bidirectional parity between the OpenAPI spec and Gin routes, including reflective field-type checking. Excellent regression net (for Keeper).
- **Structured tenant model** — DNS-syntax IDs in [internal/shared/tenant/tenant.go](../internal/shared/tenant/tenant.go), with tenant context propagated through Gin and NATS headers.
- **Postgres queries** — every repository uses parameterized queries; no string concatenation; tenant filtering present on every read/write.
- **Domain validation** — keeper and agent domain types have rich `Validate()` methods (URL parsing, regex, range checks for `temperature`, MCP transport rules).

---

## 3. Shared packages — `internal/shared/`

### 3.1 [auth/auth.go](../internal/shared/auth/auth.go)

| Sev | Finding |
|-----|---------|
| **CRITICAL** | No signature/expiry/issuer/audience verification. The package is named `auth` but performs only header parsing. Every consumer downstream trusts `X-Tenant-ID` blindly. Add a JWKS-backed validator (e.g. `github.com/MicahParks/keyfunc` or `github.com/coreos/go-oidc/v3`). |
| MED | `Claims` struct is too minimal for production (no `iss`, `aud`, `exp`, `iat`). |
| LOW | `ExtractBearerToken` uses `TrimSpace` — fine, but document the contract that signature validation happens elsewhere (it currently doesn't). |

### 3.2 [config/config.go](../internal/shared/config/config.go)

| Sev | Finding |
|-----|---------|
| MED | `_ = v.BindEnv(...)` swallows errors silently. |
| MED | Hard-coded `DATABASE_URL` / `DB_URL` fallback is fragile and component-specific; should not live in shared. |
| LOW | No `prefix == ""` guard. |

### 3.3 [tenant/tenant.go](../internal/shared/tenant/tenant.go)

Solid. Strong regex validation, immutable values, type-safe context plumbing. Suggest adding a `Parse(fullName)` companion to `FullName()`.

### 3.4 [errors/errors.go](../internal/shared/errors/errors.go)

| Sev | Finding |
|-----|---------|
| MED | Only `NewNotFound` / `NewAlreadyExists` constructors are exported; no `NewInternal`/`NewUnavailable`. |
| MED | Handlers downstream resort to `strings.Contains(err.Error(), "not found")` (see §4) — expose a public `Code()` method or define typed sentinel matches via `errors.Is`. |

### 3.5 [health/health.go](../internal/shared/health/health.go)

| Sev | Finding |
|-----|---------|
| MED | Single timeout shared across all checks — slow checker starves the rest. Apply a per-check `context.WithTimeout`. |
| MED | Cancelled checks lose their error message in the response body. |
| LOW | `HTTPChecker` uses `http.DefaultClient` rather than the shared instrumented client. |

### 3.6 [observability/](../internal/shared/observability/)

#### tracing.go
| Sev | Finding |
|-----|---------|
| **CRITICAL** | If `otlpmetricgrpc.New` fails after `otlptracegrpc.New` succeeded, the trace exporter (a live gRPC connection) is leaked — there is no cleanup path. |
| HIGH | Hard-coded `WithInsecure()` on both OTLP exporters; no env-driven TLS. |
| MED | Returned shutdown function is not idempotent. |

#### logger.go
| Sev | Finding |
|-----|---------|
| HIGH | `LogClaimsKeys` is a global mutable `[]string` — concurrent modification is a data race. Replace with a function/`sync.RWMutex`. |
| MED | `LoggingMiddleware` builds a new logger per request. |
| MED | No redaction filter for sensitive claim values. |

#### metrics.go
| Sev | Finding |
|-----|---------|
| HIGH | Errors from `meter.*Histogram(...)` are discarded with `_ =`. Failure → silently broken metric. |
| HIGH | Instruments are declared **twice** (package-level + `initInstruments()`) — risk of duplicates. |
| HIGH | `MetricsMiddleware` records `path` from `c.FullPath()`; if a route is unmapped, falls back to raw URL → unbounded cardinality. Use `"unmapped"` placeholder. |
| MED | Inconsistent meter names (`tacito-square` vs `db-pool`). |

#### gin_tracing.go / db_tracing.go / nats_tracing.go / client.go
Generally well-built. `NATSHeaderCarrier` is a small, correct piece of work. Minor issues: env-var parse errors silently swallowed in `client.go`; `db_tracing.go` does not capture row count or error type as span attributes.

### 3.7 [shutdown/shutdown.go](../internal/shared/shutdown/shutdown.go)

| Sev | Finding |
|-----|---------|
| MED | Default signal set is `os.Interrupt` only — does **not** include `syscall.SIGTERM`, which is what Kubernetes sends. Fix: default to `[]os.Signal{syscall.SIGTERM, os.Interrupt}`. |
| LOW | No re-entrancy guard on `RunHooks`. |

### 3.8 [adapters/outbound/cache/cache_adapter.go](../internal/shared/adapters/outbound/cache/cache_adapter.go)

| Sev | Finding |
|-----|---------|
| **CRITICAL** | `mockRedis` is defined in production source (not in a `_test.go` file) — it ships in every binary. Move to `cache_adapter_test.go` or a `mocks/` test package. |
| MED | `Get` does not validate that `dest` is a non-nil pointer. |
| MED | `Set` accepts non-positive TTL silently. |

### 3.9 [adapters/outbound/s3/s3_adapter.go](../internal/shared/adapters/outbound/s3/s3_adapter.go)

| Sev | Finding |
|-----|---------|
| MED | No validation of `bucket`, `endpoint`, or `key` (no `..` / leading-`/` rejection). |
| LOW | Returned URL is not URL-encoded. |

### 3.10 [ports/outbound/](../internal/shared/ports/outbound)

Clean and minimal. `ErrCacheMiss` sentinel is the right call. Add brief godoc on lifecycle expectations.

---

## 4. Keeper service — `internal/keeper/` + `cmd/keeper`

### 4.1 Bootstrap and migrations

- [bootstrap.go](../internal/keeper/bootstrap.go): **MED** — NATS, Redis, and cache health checkers are stub `func(ctx) error { return nil }`. They should reflect actual connectivity.
- **MED** — `noOpCRDCoordinator` is silently substituted when `k8sconfig.GetConfig()` fails. Should at minimum log a warning, and probably be fatal if any agent rows exist.
- [migrations.go](../internal/keeper/migrations.go) + [cmd/keeper/main.go](../cmd/keeper/main.go): **MED** — relative path `deploy/postgres/migrations` only works when the binary is launched from the repo root. Embed via `//go:embed` or require an absolute path.

### 4.2 Domain model

| Sev | Finding |
|-----|---------|
| MED | [agent.go](../internal/keeper/domain/model/agent.go) `Validate()` does not enforce the cross-field invariant: `Status ∈ {Assigned, Pending, Running, Idle}` ⇒ `CommunityID != nil` (and vice versa). |
| MED | [echo.go](../internal/keeper/domain/model/echo.go) `SanitizeMessage` silently truncates to 1000 runes — return a flag so callers can warn. |
| LOW | No `Version` / `UpdatedAt`-based optimistic-lock field on any entity. |

### 4.3 Application ports

| Sev | Finding |
|-----|---------|
| **CRITICAL** | [repositories.go](../internal/keeper/application/ports/outbound/repositories.go): `Update(ctx, *X)` methods accept the full entity with no version/etag → concurrent updates silently last-write-wins. |
| HIGH | No filter / pagination methods (`ListByStatus`, `ListByCommunity`, `ListWithLimit/Offset`). Forces the spin-wait + full-table-scan in `echo_service.waitForRunningAgents`. |
| MED | `TransactionRunner` interface exists but is not consumed by any service — multi-step writes (DB + NATS, DB + CRD) are not atomic. |
| LOW | [lifecycle_ports.go](../internal/keeper/application/ports/inbound/lifecycle_ports.go) leaks K8s-shaped fields (`Replicas int32`) into an inbound port that should be transport-neutral. |

### 4.4 Application services

| Sev | Finding |
|-----|---------|
| **CRITICAL** | [agent_service.go](../internal/keeper/application/service/agent_service.go) `Assign` / `Unassign` spawn `go func(){ s.crdCoordinator.SubmitAgentCRD(bgCtx, agent) }()` with no `WaitGroup`, no panic recovery, and no shutdown synchronization. Process exit can interrupt mid-flight K8s API call → DB shows assigned, cluster has no Deployment. Use a bounded worker pool registered with the shutdown manager, or invert to event-driven (publish, let operator pull). |
| HIGH | [echo_service.go](../internal/keeper/application/service/echo_service.go) `waitForRunningAgents` polls `agentRepo.List(ctx)` every 50 ms (20 Hz). At any non-trivial agent count this is O(n) per tick. Replace with a NATS subscription to status transitions, or with a `LISTEN/NOTIFY` pg trigger. |
| HIGH | Same file: per-agent fan-out has unbounded goroutine count (one per running agent). Cap with a `semaphore.Weighted`. |
| MED | [lifecycle_service.go](../internal/keeper/application/service/lifecycle_service.go) publishes the `started` NATS event **after** the DB write — if NATS is down, status flips with no listener notified. Consider outbox pattern or publish-then-update. |
| LOW | Agent timeout is read from a global Viper key, not per-community. |

### 4.5 HTTP adapters

| Sev | Finding |
|-----|---------|
| HIGH | [middleware.go](../internal/keeper/adapters/inbound/http/middleware.go) — `HeaderTenantResolver.Resolve` calls `tenant.New(...)` which can return `(nil, err)`; downstream handlers don't all guard for nil. |
| HIGH | Handlers do not perform an explicit `agent.TenantID == ten.FullName()` check after `repo.GetByID`. Trust is delegated to the repo's `WHERE tenant_id = $N`, which is correct, but defense-in-depth is missing. |
| MED | Multiple handlers use `strings.Contains(err.Error(), "not found")` to decide HTTP status — fragile string matching. Use typed errors (or `errors.Is(err, ErrNotFound)`). |
| MED | [llm_binding_handlers.go](../internal/keeper/adapters/inbound/http/llm_binding_handlers.go) returns `201 Created` on `Update` (should be 200/204). |
| MED | OTel span is started in handlers but `ctx` passed to services is the raw `c.Request.Context()`, not `trace.ContextWithSpan(ctx, span)` — server span is not the parent of downstream child spans. |
| LOW | Many request DTOs lack `binding:"max=N"` length caps; opens up cheap allocations. |
| LOW | No `Idempotency-Key` support on POST/PUT. |

### 4.6 Postgres adapters

| Sev | Finding |
|-----|---------|
| **CRITICAL** | [agent_repository.go](../internal/keeper/adapters/outbound/postgres/agent_repository.go) `List` performs `loadSkills(ctx, a.ID)` inside the row loop → classic N+1. Replace with a single `SELECT agent_id, skill_id FROM agent_skills WHERE agent_id = ANY($1)` and group in memory. |
| HIGH | [transaction.go](../internal/keeper/adapters/outbound/postgres/transaction.go) `ExecuteInTxOrPool` rolls back unconditionally via `defer tx.Rollback(ctx)` — if `fn` succeeds the rollback no-ops post-commit, but a panic does not re-panic, swallowing detail. Re-throw after rollback. |
| MED | No indices implied for `agents(tenant_id)`, `agents(community_id)`, `agents(status)`, `agent_skills(agent_id)` — verify the migrations under [deploy/postgres/migrations/](../deploy/postgres/migrations) include them. |
| MED | `GetByName` has no `UNIQUE(tenant_id, name)` constraint (verify in migrations). |
| LOW | `community_repository.go` returns `nil` slice from empty `List`; other repos normalize to `make([]T, 0)` — be consistent. |

### 4.7 CRD coordinator

| Sev | Finding |
|-----|---------|
| HIGH | [crd_coordinator.go](../internal/keeper/adapters/outbound/crd/crd_coordinator.go) `ResolveAndSynthesizeSystemPrompt` runs synchronously during `SubmitAgentCRD` — slow Postgres or template lookup blocks K8s API call. Consider caching templates. |
| HIGH | MCP client config lookup is done in a per-client loop → batch-load with `GetByIDs`. |
| MED | NATS provisioning-event publishes silently log-and-continue on error — use JetStream with persistence or an outbox. |
| MED | Synthesized JSON is not size-checked against K8s' ~1 MiB object limit; large skill catalogues could overflow. |
| LOW | `namespace: "tacito"` is hard-coded. |

### 4.8 NATS outbound (community broadcaster)

| Sev | Finding |
|-----|---------|
| HIGH | [community_broadcaster.go](../internal/keeper/adapters/outbound/nats/community_broadcaster.go) request-reply has no minimum timeout if the caller's context has no deadline. |
| MED | No metric for publish latency / failure count. |
| LOW | No payload-size guard before publish. |

---

## 5. Operator service — `internal/operator/` + `cmd/operator`

### 5.1 Reconciler — [reconciler.go](../internal/operator/adapters/inbound/reconciler.go)

| Sev | Finding |
|-----|---------|
| **CRITICAL** | No finalizer registered on `TacitoAgent`. When the CR is deleted, reconciliation just exits on `IsNotFound`; child Deployments / Services are left to garbage-collection by `OwnerReferences` — which **is** correct for owned resources, but any external-side effects (NATS broadcast, status-cleanup, persisted metrics) are lost. Add a finalizer for cleanup hooks even if K8s GC handles owned objects. |
| HIGH | Errors are returned as `ctrl.Result{}, err` with no `RequeueAfter` strategy. Transient API conflicts will retry with controller-runtime's default backoff, but rate-limit it explicitly for clarity. |
| HIGH | No `ResourceVersion` check before `Status().Update()` — status conflicts under multi-replica operator (or rapid reconciles) silently lose updates. |
| MED | Metrics are initialized both in `init()` and in `InitReconcilerMetrics()` — pick one. |
| MED | `updateActiveAgentsMetric` runs on every reconcile (full list) — move to a 30-second background ticker. |

### 5.2 Reconcile service — [reconcile_service.go](../internal/operator/application/service/reconcile_service.go)

| Sev | Finding |
|-----|---------|
| **CRITICAL** | `existingDep.Spec = dep.Spec` (and the equivalent for `Service`) overwrites all fields. `Deployment.Spec.Selector` is **immutable** in K8s — first update will return `Invalid value: ... field is immutable`. Use `apiequality.Semantic.DeepEqual` on the diffable subset, or a strategic-merge / server-side apply (`client.Apply`). |
| **CRITICAL** | For `Service`, only `ClusterIP` is preserved; `SessionAffinity`, `ExternalTrafficPolicy`, etc. would be wiped if ever set externally. |
| HIGH | All errors are returned uniformly — no classification of transient (requeue) vs terminal (event + give up). |
| HIGH | Hardcoded fallbacks (`defaultNatsURL = "nats://tacito-infra-nats:4222"`) silently mask missing config. Prefer fail-fast. |
| MED | No spec validation prior to building objects — invalid agent name → invalid env var name → confusing pod failure. |
| MED | `imagePullPolicy` always `IfNotPresent`; no override path for canary. |

### 5.3 CRD types — [pkg/kubernetes/apis/tacito/v1alpha1/tacitoagent_types.go](../pkg/kubernetes/apis/tacito/v1alpha1/tacitoagent_types.go)

| Sev | Finding |
|-----|---------|
| HIGH | `Temperature *string` validated by regex — should be `*resource.Quantity` or numeric `*float32` with `+kubebuilder:validation:Minimum/Maximum`. |
| MED | `MaxTokens` capped at 8192 — too low for modern models (GPT-4o supports 128k context, 16k output). |
| MED | `SystemPrompt` has no `maxLength`. |
| MED | `MCPClients` has no `maxItems`. |

### 5.4 Operator main / bootstrap

| Sev | Finding |
|-----|---------|
| HIGH | `cmd/operator/main.go` launches manager + HTTP server in goroutines without `errgroup` — early-failure of one is not a hard exit. |
| MED | `k8sconfig.GetConfig()` has no timeout. |

---

## 6. Agent service — `internal/agent/` + `cmd/agent`

### 6.1 Bootstrap

| Sev | Finding |
|-----|---------|
| HIGH | Unknown brain `provider` falls through to OpenAI silently — should be a fatal error. |
| HIGH | If brain does not implement `Embedder`, Qdrant init silently degrades to a no-op. |
| MED | Malformed `TS_AGENT_MCP_CLIENTS` JSON logs and continues with empty clients. |

### 6.2 Domain model

Generally clean. Note:
- **MED** `LTMEntry.Validate` does not check `len(Embedding) == expectedDim` — only catches at upsert time.
- **MED** `MemoryEntry` timestamps are unbounded.

### 6.3 Cognitive engine + message processor

| Sev | Finding |
|-----|---------|
| **CRITICAL** | [message_processor.go](../internal/agent/application/service/message_processor.go) `triggerMemoryConsolidation` launches `go func()` with `context.WithoutCancel(ctx)`. No `WaitGroup`, no shutdown coordination, no timeout. On heavy traffic these accumulate; on shutdown they are killed mid-Qdrant-write. Track them explicitly and make `Stop()` block until they drain (with a deadline). |
| HIGH | `cognitive_engine.GenerateStream` is a stub that calls `Generate` and emits one chunk — defeats the purpose of streaming and breaks any caller expecting incremental token delivery. |
| HIGH | Built-in tool name shadowing: MCP tool named `recall_memory` would override the built-in. Namespace tool names (`builtin:recall_memory`, `mcp:foo`). |
| HIGH | System prompt is JSON-unmarshalled with errors discarded silently; malformed config → silently empty skill list. |
| HIGH | On STM `Append` failure the user message is dropped and reasoning continues with empty history — degrades silently. |
| MED | History fallback on read failure is `[userEntry]` (no prior turns). |
| MED | Max-reasoning-step exceeded path returns a vague message instead of a clear error. |

### 6.4 NATS echo subscriber — [echo_subscriber.go](../internal/agent/adapters/inbound/nats/echo_subscriber.go)

| Sev | Finding |
|-----|---------|
| **CRITICAL** | No `defer recover()` in the message handler. A single panic (e.g. in JSON parse or LLM call) terminates the goroutine; depending on subscriber setup the subscription may stop processing further messages silently. |
| HIGH | Malformed payload returns `nil` with no NACK and no metric — message is acked-as-success but never processed. |
| HIGH | No timeout on `msg.Respond` — slow brain wastes resources after requestor has timed out. |
| MED | Thread ID falls back to `"thread-" + communityID`, conflating concurrent conversations. |

### 6.5 Outbound adapters

#### Redis STM — [memory_adapter.go](../internal/agent/adapters/outbound/redis/memory_adapter.go)
- HIGH — corrupted JSON entries are silently skipped on `Get` (no log, no metric).
- MED — TTL is per-thread-list, not per-entry.
- MED — default go-redis pool sizing not tuned.

#### Qdrant LTM — [ltm_adapter.go](../internal/agent/adapters/outbound/qdrant/ltm_adapter.go)
- **CRITICAL** — single shared collection across all tenants; isolation depends entirely on a manually-built filter. Use one collection per tenant (or per tenant prefix).
- **CRITICAL** — embedding dimension mismatch is not validated client-side; surfaces as gRPC error with poor message.
- HIGH — `threshold := float32(0.7)` is hard-coded.
- HIGH — `Search` has no pagination / cursor.
- HIGH — `buildSearchFilter` is intricate, has no unit tests visible — high risk of cross-tenant leak in edge case.
- MED — payload field names are magic strings; promote to constants.

#### OpenAI / Ollama adapters
- **CRITICAL** — both log full request bodies (`Interface("request_body", req)`) at debug level. With `LOG_LEVEL=debug` in production this dumps user prompts and any embedded secrets to stdout / collector. Strip or redact.
- HIGH — retry timeout is per-attempt, not total; effective timeout = N × per-attempt.
- HIGH — no allow-list for retryable errors; 401/403 will retry uselessly.
- MED — `temperature == 0` cannot be explicitly set (the `> 0` guard sends nothing).
- MED — `GenerateStream` is stubbed (same issue noted in §6.3).

#### Resiliency — [circuit_breaker.go](../internal/agent/adapters/outbound/resiliency/circuit_breaker.go)
- HIGH — half-open admits exactly one request; concurrent callers serialize on the mutex with no fairness.
- HIGH — half-open has no failure counter / timeout — one failure flips back to open immediately, but rapid oscillation is possible.
- HIGH — `beforeExecute` and `afterExecute` are separate locked sections (TOCTOU window).
- MED — no logs / metrics on state transitions.

#### MCP adapter — [mcp_adapter.go](../internal/agent/adapters/outbound/mcp/mcp_adapter.go)
- HIGH — empty `AllowedTools` silently denies all (probably intended, but make it explicit).
- HIGH — SSE transport has no reconnect logic for transient network failures.
- HIGH — execution errors on the breaker path return `{"error":"..."}` JSON instead of a Go `error`.
- MED — `AllowedTools` matching is case-sensitive; mismatches silently drop tools.

---

## 7. BFF service — `internal/bff/` + `cmd/bff`

[internal/bff/bootstrap.go](../internal/bff/bootstrap.go) currently exposes only `/healthz` and `/readyz`. Per [docs/architecture/overview.md](../docs/architecture/overview.md), BFF is supposed to validate JWTs, aggregate Keeper REST calls, and serve React UIs. None of that is implemented.

| Sev | Finding |
|-----|---------|
| **CRITICAL** | No proxy / aggregation logic. |
| **CRITICAL** | No JWT auth middleware. |
| **CRITICAL** | No `/metrics` (other services have it). |
| HIGH | No CORS middleware — browser clients will be blocked. |
| HIGH | No security headers (`X-Content-Type-Options`, `X-Frame-Options`, `HSTS`, CSP). |
| HIGH | No request body size limit; vulnerable to large-payload DoS. |
| HIGH | No request timeout middleware. |
| HIGH | No rate limiter. |
| MED | `cmd/bff/main.go` `http.Server` has no `ReadTimeout` / `WriteTimeout` / `IdleTimeout` / `MaxHeaderBytes`. |
| MED | No upstream-readiness checker — `/readyz` returns 200 even if Keeper is unreachable. |

`internal/bff/bootstrap_test.go` only covers the two health endpoints; security tests, header propagation, and JWT scenarios are absent.

---

## 8. Cross-service consistency

| Aspect | BFF | Keeper | Agent | Operator |
|--------|-----|--------|-------|----------|
| Viper config (`TS_*`) | ✓ | ✓ | ✓ | ✓ |
| zerolog | ✓ | ✓ | ✓ | ✓ |
| OTel tracer init | ✓ | ✓ | ✓ | ✓ |
| `shutdown.Manager(30s)` | ✓ | ✓ | ✓ | ✓ |
| Gin HTTP server | ✓ | ✓ | ✓ | n/a (controller-runtime) |
| `/healthz` + `/readyz` | ✓ | ✓ | ✓ | via mgr |
| `/metrics` | ✗ | ✓ | partial | via mgr |
| Real readiness checkers | ✗ | partial (stubs) | ✓ | via mgr |
| Contract test | ✗ | ✓ | ✗ | ✗ |

Recommendations:
- Promote a small shared bootstrap helper (`shared/bootstrap`) that wires logger + tracer + shutdown + `/metrics` + standard middlewares — currently each `main.go` reimplements ~100 nearly-identical lines.
- Standardize the error response shape (`{error, code, status}`) and put it in `shared/errors`.
- Standardize `syscall.SIGTERM` handling at the shutdown manager level (see §3.7).

---

## 9. Build, deps, tests

- **`go.mod`** is on Go 1.26.3 with current versions of k8s.io (v0.36.x), OTel (v1.44.x), pgx (v5.9.x), NATS (v1.52.x). No `replace` directives. `gin v1.12.0` is bleeding-edge — consider pinning to the latest patch of v1.11.x for stability.
- No Dependabot / Renovate / SBOM workflow visible — add `.github/dependabot.yml` for weekly module + GH-action updates.
- **Test coverage gaps** (consolidated):
  - Operator: no finalizer / requeue / status-conflict tests.
  - Reconcile service: no tests for spec-merge correctness on update.
  - NATS echo subscriber: no tests for panic recovery, malformed payload, reply timeout.
  - Qdrant adapter: no tests for tenant isolation in `buildSearchFilter`.
  - Circuit breaker: no concurrency tests, no half-open behavior.
  - LLM adapters: no test asserting that prompts are **not** logged.
  - BFF: only health probes covered.
  - Contract test: only Keeper — extend to BFF (once routes exist), Agent, Operator.
- **OWASP Top 10 quick map**:
  - A01 Broken access control — see C1, missing tenant ownership double-check (§4.5).
  - A02 Crypto failures — `WithInsecure()` on OTLP, no TLS toggle (§3.6).
  - A04 Insecure design — BFF skeleton (§7).
  - A05 Security misconfig — no security headers / CORS (§7).
  - A07 Auth failures — no JWT validation (§3.1).
  - A09 Logging failures — prompts logged (§6.5).

---

## 10. Recommended remediation phases

### Phase 0 — block production (≤ 1 sprint)
1. Implement real OIDC/JWT middleware (`shared/auth`); wire into Keeper + BFF.
2. Strip / redact request bodies from LLM adapter logs.
3. Add `defer recover()` (with metric increment) in NATS subscriber.
4. Move `mockRedis` to a `_test.go` file.
5. Add `syscall.SIGTERM` to shutdown defaults.
6. Fix OTel tracer init resource leak on metric-exporter failure.

### Phase 1 — correctness & isolation (1–2 sprints)
1. Per-tenant Qdrant collections (or hardened filter + tests + Qdrant RBAC).
2. Operator: add finalizer, fix Deployment/Service update with strategic merge / server-side apply, add requeue strategy.
3. Repository update methods: introduce a `Version int` column + check-and-set.
4. Replace echo `waitForRunningAgents` poll loop with NATS notification.
5. Goroutine supervision: `WaitGroup` + shutdown integration for `agent_service.Assign` background work and `triggerMemoryConsolidation`.
6. Fix N+1 in `agent_repository.List` via batched skill load.

### Phase 2 — BFF, hardening, observability (2–4 sprints)
1. Implement BFF as a real gateway: reverse proxy, JWT validation, CORS, security headers, request size limit, request timeout, rate limit, `/metrics`, contract tests.
2. Standardized error response across services; replace `strings.Contains(err.Error(), "not found")`.
3. Streaming LLM responses (real `GenerateStream`).
4. Circuit breaker observability + correctness fixes (half-open throttling, state-change logs).
5. Dependabot / SBOM / dependency scanning workflow.

### Phase 3 — quality & scale (continuous)
1. Pagination on all `List` ports.
2. Per-tenant/per-community config overrides for timeouts and thresholds.
3. Embed pg migrations via `//go:embed`.
4. Extend contract tests to Agent, Operator, BFF.
5. Integration test for tenant-isolation and multi-replica reconciler scenarios.

---

## 11. Severity rollup

| Sev | Count (approx) |
|-----|----------------|
| CRITICAL | 12 |
| HIGH | 35 |
| MED | 50+ |
| LOW / NIT | 25+ |

Total Go source reviewed: ≈168 files. Estimated implementation effort to reach Phase 1 done: medium (most fixes are localized and well-scoped thanks to the hexagonal layering).
