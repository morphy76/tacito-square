# SPEC-NFR-CACHE: Redis Infrastructure Cache

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-NFR-CACHE                     |
| Status        | ACCEPTED                           |
| Component     | keeper, shared                     |

## Specification

1. Redis MAY be used as a general-purpose infrastructure cache beyond agent short-term memory.
2. Use cases:
   - **Keeper**: Cache prompt templates and skill descriptors to avoid repeated PostgreSQL reads
   - **Keeper**: Cache community config and quota counters
   - **Agent**: Cache resolved MCP tool schemas to avoid repeated discovery
   - **BFF**: Cache OIDC JWKS responses and token introspection results
3. A shared `Cache` port MUST be defined:
   - `Get(ctx, key string, dest interface{}) error`
   - `Set(ctx, key string, value interface{}, ttl time.Duration) error`
   - `Invalidate(ctx, key string) error`
4. Cache keys MUST be prefixed with component name: `ts:{component}:cache:{key}`.
5. The Redis adapter MUST distinguish between STM and cache usage via key namespacing (same Redis instance is acceptable for dev; separate instances configurable for production).
6. Cache TTLs MUST be configurable per use case.
7. Cache eviction policies MUST be configurable per use case.
8. Cache operations MUST be thread-safe.

## Acceptance Criteria

1. `Cache.Set` + `Cache.Get` round-trip with JSON serialization
2. TTL expiry works correctly
3. Key namespace isolation between STM and cache
4. Cache miss returns `ErrCacheMiss` (sentinel error)
5. Invalidation removes key immediately
