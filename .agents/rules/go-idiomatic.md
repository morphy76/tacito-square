---
trigger: glob
description: Enforce Go idiomatic coding style, patterns, and efficiency rules across the project.
globs: **/*.go
---

# Go Idiomatic Coding Rules

These rules apply to every `.go` file in the project. They enforce standard Go community conventions,
project-specific patterns, and efficiency guidelines. Violations must be corrected before completing
any task involving Go code.

---

## 1. Naming Conventions

- **Packages:** lowercase, single-word nouns (`model`, `service`, `postgres`). Never `utils`, `helpers`, `common`.
- **Exported identifiers:** `MixedCaps`; unexported: `camelCase`. Never underscores or Hungarian notation.
- **Interfaces:** suffix with `-er` when the interface describes a single behaviour (`AgentRepository`, `EventPublisher`). Multi-method interfaces use descriptive domain names.
- **Acronyms:** keep consistent case — `ID`, `URL`, `HTTP`, `JSON`, `CRD`, `LLM`, `NATS`, `TTL`. Never `Id`, `Url`, `Http`.
- **Constructors:** always `New<Type>(deps...) *Type`. Return the concrete type, not the interface.
- **Boolean variables/fields:** use `is`, `has`, `ok` prefix only when it meaningfully disambiguates (`isCRDActive`, `hasSpan`).
- **Error variables:** `var ErrAgentNotFound = errors.New("agent not found")`. Exported sentinel errors for domain conditions.

---

## 2. Error Handling

- **Always handle errors explicitly.** Never discard with `_` except for intentional fire-and-forget goroutine results (document why).
- **Wrap errors with context** using `fmt.Errorf("operation context: %w", err)`. The message prefix must identify the operation, not repeat the caller name.
- **Sentinel errors:** use `errors.Is` / `errors.As` for comparison. Never string-match error messages.
- **Early return on error** (guard clauses). Avoid deeply nested `if err == nil { ... }` chains.
- **Domain errors vs. infrastructure errors:** return raw infrastructure errors wrapped with domain context; never leak raw SQL/pgx/redis errors to the application or inbound layers.
- **pgx `ErrNoRows`:** always convert to a domain-meaningful error (e.g., `"agent not found: <id>"`).
- **Do not use `panic`** in production paths. Reserve it for programmer errors in `init()` or test helpers.

```go
// CORRECT
if err := s.repo.Create(ctx, agent); err != nil {
    return fmt.Errorf("create agent: %w", err)
}

// WRONG — no context, leaks raw error
return s.repo.Create(ctx, agent)
```

---

## 3. `context.Context` Propagation

- **Always accept `ctx context.Context` as the first parameter** of any function that performs I/O, calls external services, spawns goroutines, or may need cancellation.
- **Never store context in a struct.** Pass it explicitly through the call chain.
- **Background goroutines:** detach from the request context with `context.Background()`, then re-inject tenant and OTel span:
  ```go
  bgCtx := context.Background()
  bgCtx = tenant.ContextWithTenant(bgCtx, ten)
  if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
      bgCtx = trace.ContextWithSpan(bgCtx, span)
  }
  go func() { _ = s.doWork(bgCtx) }()
  ```
- **Propagate `context.Context` across package and RPC boundaries.** Every outbound port call, pgx query, Redis command, NATS publish, and OTel span creation must receive the active `ctx`.
- **Respect cancellation:** check `ctx.Err()` inside loops or before expensive operations.

---

## 4. Interface Design (Ports & Adapters)

- **Define interfaces at the consumer site** (`application/ports/outbound/`), not in the implementing package.
- **Keep interfaces small:** prefer single-method interfaces or logically minimal groups. Do not add methods to an interface until there is a real consumer need.
- **Accept interfaces, return concrete types** in constructors. Application services receive port interfaces; adapters return their concrete type.
- **Never embed `interface{}` or `any` in domain interfaces** — use typed parameters and return values.
- **Mock generation:** write manual mocks in `_test.go` files using struct literals satisfying the interface; do not import mock-generation frameworks unless testify/mock is already a dependency.

---

## 5. Struct & Type Design

- **Value receivers** for small, immutable types and domain validation (`func (a Agent) Validate() error`).
- **Pointer receivers** for all service/handler/repository types and any method that mutates state.
- **`omitempty` on optional JSON fields.** Never omit the tag on exported struct fields that participate in serialization.
- **Use typed constants (`type Status string`)** for enumerations. Never raw string comparisons across packages.
- **Avoid struct embedding** unless it genuinely represents an "is-a" relationship. Prefer composition via named fields.
- **Zero-value usefulness:** design structs so the zero value is meaningful or obviously invalid (use pointer fields for optional/nullable domain concepts).

---

## 6. Concurrency

- **Goroutine ownership:** the goroutine that creates a channel owns its close. Document goroutine lifecycles with comments.
- **Use `context.Context` to signal goroutine termination.** Never use `time.Sleep` as a synchronization primitive.
- **`sync.WaitGroup`** for fan-out patterns. `errgroup.Group` (from `golang.org/x/sync/errgroup`) for parallel work that can return errors.
- **Avoid shared mutable state.** Prefer passing data through channels or returning values. If a mutex is required, keep its scope as narrow as possible; document what it protects.
- **Readiness checks in parallel:** use `errgroup` with a shared `ctx` to fan-out dependency checks concurrently (required by `SPEC-NFR-HEALTH`).
- **Do not leak goroutines.** Every goroutine must either complete normally, be cancelled by a `context.Context`, or be tracked by a `WaitGroup`.

---

## 7. Testing

- **Test files in the same package** (`package service`, not `package service_test`) unless testing the public API surface.
- **Use `testify/assert` and `testify/require`** for assertions. `require` for setup failures that make the rest of the test meaningless; `assert` for non-fatal checks.
- **Table-driven tests** for functions with multiple input variants:
  ```go
  tests := []struct {
      name    string
      input   T
      want    T
      wantErr bool
  }{...}
  for _, tt := range tests {
      t.Run(tt.name, func(t *testing.T) { ... })
  }
  ```
- **HTTP handler tests:** always use `gin.SetMode(gin.TestMode)` + `httptest.NewRecorder()` + `engine.ServeHTTP(w, req)`. Never instantiate handlers without the full Gin middleware stack.
- **Integration tests:** use `testcontainers-go` with `postgres` and `redis` modules. Tag integration tests with `//go:build integration`.
- **No `time.Sleep` in tests.** Use channels, `sync.WaitGroup`, or `testify/mock` call expectations to synchronize.
- **Avoid `t.Parallel()` unless the test is verified to be safe for concurrent execution** with shared containers or resources.

---

## 8. Logging (zerolog)

- **Use the package-level `zerolog.Logger`** retrieved from context or injected via constructor. Never use `log.Println` or `fmt.Print` for application logging.
- **Log levels:**
  - `Error`: unexpected, blocking conditions.
  - `Warn`: recoverable or non-blocking degraded conditions.
  - `Info`: business-level state changes and completions (public methods).
  - `Debug`: branching decisions, entry points of public methods.
  - `Trace`: private method details, low-level diagnostics.
- **Inject OTel trace context into every log entry** when an active span is present:
  ```go
  log.Debug().Str("trace_id", span.SpanContext().TraceID().String()).Msg("...")
  ```
- **Never log secrets, PII, or full request bodies.** Log IDs and structural metadata only.
- **Use structured fields** (`Str`, `Err`, `Stringer`, `UUID`) — never string interpolation inside `.Msg()`.

---

## 9. Package & Import Organization

- **Import grouping** (goimports order, enforced by `goimports` / `gofmt`):
  1. Standard library
  2. Third-party dependencies
  3. Internal project packages (`github.com/morphy76/tacito-square/...`)
- **No circular imports.** Domain → nothing; Application → Domain; Adapters → Application + Domain.
- **Alias imports only when disambiguation is required** (e.g., `sharedports "github.com/morphy76/tacito-square/internal/shared/ports/outbound"`). Document the alias with a comment if not obvious.
- **No dot imports** (`import . "pkg"`) outside of test files.

---

## 10. Performance & Efficiency

- **Pre-allocate slices** when the length is known: `make([]T, 0, n)`.
- **Use `strings.Builder`** for multi-step string construction. Avoid repeated `+` concatenation in loops.
- **Avoid unnecessary allocations in hot paths:** prefer value types over pointers for small structs passed by value.
- **Re-use `pgxpool.Pool`** — never create ad-hoc connections. Always call `GetExecutor(ctx, pool)` to honour transaction context.
- **Close resources in `defer`** immediately after acquisition. Document if `defer` is intentionally skipped for performance reasons.
- **Avoid `reflect` in production code.** Use type switches or interface methods instead.
- **`json.Marshal`/`json.Unmarshal`:** reuse encoding for repeated serialization of the same type. Consider `json.RawMessage` for pass-through payloads.

---

## Developer Checklist

- [ ] All exported identifiers follow `MixedCaps`; acronyms are consistently cased (`ID`, `LLM`, `URL`).
- [ ] Every error is wrapped with `fmt.Errorf("context: %w", err)` and handled explicitly.
- [ ] `context.Context` is the first parameter of every I/O function and propagated to all callees.
- [ ] Background goroutines re-inject tenant context and OTel span before dispatching work.
- [ ] Interfaces are defined at the consumer site and kept minimal.
- [ ] HTTP handler tests run through `gin.ServeHTTP` in `gin.TestMode`.
- [ ] Log levels match the observability rule: `error`/`warn`/`info`/`debug`/`trace`.
- [ ] Imports are grouped: stdlib → third-party → internal.
- [ ] No circular imports; domain layer imports nothing from application or adapters.