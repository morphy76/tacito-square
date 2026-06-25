# TASK-M7.2-T8: BFF UI Index Proxy & Redis Caching

| Field       | Value                                                              |
|-------------|--------------------------------------------------------------------|
| Task ID     | TASK-M7.2-T8                                                      |
| Spec        | SPEC-FR-M7.2                                                       |
| Boundary    | BFF (`internal/bff/`)                                              |
| Status      | VERIFIED                                                           |
| Depends On  | SPEC-FR-M7.1, TASK-M7.2-T6, TASK-M7.2-T7                           |

## Objective

Implement a reverse proxy handler inside the Go BFF for serving the React SPA's `index.html` under the `/ui/` and `/` path namespaces. The handler must:
1. Enforce active session authentication before serving the page (redirecting unauthenticated users to `/api/v1/auth/login`).
2. Proxy requests for the entry HTML to the internal `ui-configurator` Nginx service (e.g. `http://ui-configurator/index.html`).
3. Cache the retrieved `index.html` content in **Redis** with a short TTL (e.g., 5 minutes) or conditional ETag validation. This ensures that new BFF replicas and scaled instances serve the page instantly from memory without adding query load to Nginx.

## Files

| File | Action |
|------|--------|
| `internal/bff/adapters/inbound/http/ui_proxy_handler.go` | NEW |
| `internal/bff/bootstrap.go` | MODIFY |
| `internal/bff/bootstrap_test.go` | MODIFY |

## RED Phase

1. **Proxy & Redis Cache Tests**:
   - Write unit tests in `bootstrap_test.go` verifying the proxy logic:
     - **Unauthenticated Redirect**: A request to `/ui/` without a session cookie is redirected to `/api/v1/auth/login` (status `302 Found`).
     - **Proxy and Cache Invalidation**: A request to `/ui/` with a valid session cookie queries the mock Nginx service, receives `index.html`, stores it in a mock Redis client under key `bff:cache:ui:index_html`, and returns `200 OK` with the matching body.
     - **Redis Serve**: A second request to `/ui/` serves the index directly from the mock Redis cache, verifying that the mock Nginx server is *not* queried again.
   - Run tests (`make test`) — must fail because the proxy handler and Redis caching do not exist.

## GREEN Phase

1. **Implement UI Proxy Handler**:
   - Create `internal/bff/adapters/inbound/http/ui_proxy_handler.go`.
   - Implement `ServeUIIndex(c *gin.Context)`:
     - Verify user session (reusing the authentication context set by the middleware).
     - Check Redis for key `bff:cache:ui:index_html`.
     - If found: Stream the cached HTML from Redis with header `Content-Type: text/html; charset=utf-8` and `Cache-Control: no-cache`.
     - If not found in Redis:
       - Send a request to `http://ui-configurator/index.html` (resolving host from configuration, default `http://ts-ui-configurator`).
       - If Nginx returns `200 OK`, store the HTML content in Redis under the key `bff:cache:ui:index_html` with a TTL of 5 minutes.
       - Stream the response to the user.
       - If Nginx is unreachable or returns an error, return a fallback offline message or the last known cached copy if available.

2. **Register Routes in Bootstrap**:
   - In `internal/bff/bootstrap.go`, register the UI proxy handler for `GET /` and `GET /ui/*` (ensuring it handles fallback routes for client-side SPA routing).
   - Ensure the Ingress routes `/ui/assets/*` directly to Nginx to prevent loading the BFF with static file transfers.

3. **Verify tests**:
   - Run `make test` and confirm all tests pass.

## REFACTOR Phase

- Audit the Redis cache key schema to ensure consistency.
- Ensure appropriate timeouts (e.g. 2 seconds) on the internal HTTP request to the Nginx service to prevent blocking the handler if Nginx is slow.
