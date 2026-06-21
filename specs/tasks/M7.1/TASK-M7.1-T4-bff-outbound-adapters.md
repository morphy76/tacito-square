# TASK-M7.1-T4: BFF Outbound Adapters (`internal/bff/adapters/outbound/`)

| Field       | Value                                                  |
|-------------|--------------------------------------------------------|
| Task ID     | TASK-M7.1-T4                                           |
| Spec        | SPEC-FR-M7.1                                           |
| Boundary    | BFF Outbound Adapters — `internal/bff/adapters/outbound/` |
| Status      | IMPLEMENTED                                            |
| Depends On  | TASK-M7.1-T2                                           |

## Objective

Implement the four driven adapters that satisfy the outbound port interfaces: Redis session repository, Zitadel OIDC HTTP client, Keeper HTTP proxy client, and the backend SSE HTTP proxy client. All adapters must include circuit breakers and configurable timeouts per SPEC-NFR-CLOUD.

## Files

| File | Action |
|------|--------|
| `internal/bff/adapters/outbound/redis_session_store.go` | NEW |
| `internal/bff/adapters/outbound/redis_session_store_test.go` | NEW |
| `internal/bff/adapters/outbound/oidc_http_client.go` | NEW |
| `internal/bff/adapters/outbound/oidc_http_client_test.go` | NEW |
| `internal/bff/adapters/outbound/keeper_http_client.go` | NEW |
| `internal/bff/adapters/outbound/keeper_http_client_test.go` | NEW |
| `internal/bff/adapters/outbound/backend_sse_client.go` | NEW |
| `internal/bff/adapters/outbound/backend_sse_client_test.go` | NEW |

## RED Phase

Write integration tests using testcontainers-go and HTTP test servers:

**`redis_session_store_test.go`** (testcontainers-go with Redis):
- `TestRedisSessionStore_SaveAndGet`: Save a `Session`, retrieve by session ID, assert all fields match including `TenantID`.
- `TestRedisSessionStore_Expiry`: Save a session with a 1-second TTL; wait 2 seconds; assert `Get` returns `ErrSessionNotFound`.
- `TestRedisSessionStore_DeleteByUserID`: Save two sessions for the same `UserID`; call `DeleteByUserID`; assert both `Get` calls return `ErrSessionNotFound`.

**`oidc_http_client_test.go`** (httptest mock server):
- `TestOIDCClient_ExchangeCode_Success`: Stand up a mock token endpoint; assert returned `TokenSet` has correct `AccessToken` and `ExpiresIn`.
- `TestOIDCClient_FetchUserInfo_Success`: Stand up a mock UserInfo endpoint; assert `UserInfoPayload` has correct `TenantID` and `Sub`.
- `TestOIDCClient_ValidateLogoutToken_ValidToken`: Provide a signed Logout Token (using test keys); assert `sub` is correctly extracted.
- `TestOIDCClient_CircuitBreaker`: Make the mock OIDC endpoint fail 6 consecutive times; assert the circuit breaker opens and subsequent calls short-circuit without hitting the server.

**`keeper_http_client_test.go`** (httptest mock server):
- `TestKeeperClient_Ping_Success`: Stand up a mock Keeper `/healthz`; assert `Ping` returns `nil`.
- `TestKeeperClient_Timeout`: Stand up a mock server that never responds; assert `Ping` returns an error within the configured deadline.

**`backend_sse_client_test.go`** (httptest mock SSE server):
- `TestBackendSSEClient_StreamEvents_Success`: Open an SSE stream from a mock server emitting 3 events; assert all 3 raw byte payloads are received on the returned channel.
- `TestBackendSSEClient_ContextCancellation`: Cancel context mid-stream; assert the returned channel is closed gracefully.

Run `make test` — must fail (RED).

## GREEN Phase

1. **`redis_session_store.go`** — `RedisSessionStore` implementing `outbound.SessionStore`:
   - Use `go-redis` client.
   - Serialize/deserialize `Session` via `encoding/json` with the key pattern `bff:session:<sessionID>`.
   - For `DeleteByUserID`: use a secondary Redis Set `bff:user-sessions:<userID>` to maintain a mapping of all session IDs per user; iterate and delete each.
   - Use pipeline for atomic multi-key operations where possible.

2. **`oidc_http_client.go`** — `OIDCHTTPClient` implementing `outbound.OIDCProvider`:
   - Use `github.com/zitadel/oidc/v3` for OIDC discovery, token exchange, and Logout Token validation.
   - Wrap all outbound calls with a circuit breaker (use `sony/gobreaker` or equivalent approved library) and Viper-configured timeout via `context.WithTimeout`.
   - Implement exponential backoff with jitter on transient 5xx errors.

3. **`keeper_http_client.go`** — `KeeperHTTPClient` implementing `outbound.KeeperClient`:
   - Issue `GET <keeper_base_url>/healthz` for `Ping`.
   - Propagate `Authorization: Bearer <token>` header from the calling context.
   - Propagate OTel W3C traceparent headers on all outbound requests.

4. **`backend_sse_client.go`** — `BackendSSEClient` implementing `outbound.BackendEventClient`:
   - Establish HTTP GET connection to the backend's SSE endpoint, propagating `Authorization` and OTel trace headers.
   - Scan the response body line-by-line; forward raw event byte frames on the returned channel.
   - Close the channel when the context is cancelled or the server closes the connection.

Run `make test` — tests must pass (GREEN).

## REFACTOR Phase

- Confirm all four structs satisfy their respective port interfaces with a compile-time assertion (`var _ outbound.SessionStore = (*RedisSessionStore)(nil)` etc.).
- Verify circuit breaker state transitions are logged at `warn` level using zerolog.
- Confirm the Redis store never stores raw tokens in unencrypted keys; confirm the key namespace is configurable via Viper (`bff.redis.key_prefix`).
- Verify OTel span attributes are added to all outbound calls (service name, URL, HTTP status).
