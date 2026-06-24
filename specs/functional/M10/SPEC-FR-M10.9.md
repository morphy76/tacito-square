# SPEC-FR-M10.9: HTTP Cache Headers for OpenAPI Specification Endpoints

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M10.9                               |
| Status        | IMPLEMENTED                                 |
| Milestone     | M10                                         |
| Component     | bff, keeper                                 |
| Depends On    | SPEC-FR-M10.2                               |
| Supersedes    | none                                        |

## Context

The OpenAPI specification payloads (`openapi.json`) served by the BFF and Keeper components are large and static for the duration of a deployment.
To reduce bandwidth, latency, and server CPU usage, standard HTTP caching headers should be added, allowing clients and downstream proxies (CDNs, gateways, reverse proxies) to cache the specification and validate updates using conditional requests.

## Specification

1. The `/openapi.json` and `/ui/openapi.json` endpoints in BFF and `/openapi.json` in Keeper MUST strongly leverage HTTP cache infrastructure by adding the following headers:
   * `Cache-Control`: `public, max-age=3600, must-revalidate`
   * `ETag`: A strong entity tag calculated using the SHA-256 hash of the JSON response payload.
2. The hash for the ETag SHOULD be computed once during server initialization or startup, since the payload remains unchanged during the lifecycle of the process.
3. The server MUST intercept incoming conditional requests containing the `If-None-Match` header. If the header matches the computed ETag, the server MUST return `304 Not Modified` with an empty response body.

## Acceptance Criteria

1. Requesting `GET /openapi.json` or `GET /ui/openapi.json` returns `Cache-Control` header set to `public, max-age=3600, must-revalidate`.
2. The response includes an `ETag` header containing the SHA-256 hash of the payload wrapped in double quotes.
3. A subsequent request with `If-None-Match` set to the returned ETag value yields a `304 Not Modified` response with no body content.
4. All existing tests pass, and OpenAPI endpoints continue to work.

## Test Plan

### Automated Tests
1. **Unit Tests:**
   * Modify or add tests in `internal/bff/bootstrap_test.go` and `internal/keeper/bootstrap_test.go` to assert caching headers.
   * Send requests with valid `If-None-Match` values and assert they receive `304 Not Modified`.
2. **Suite execution:**
   * Execute:
     ```bash
     make test
     ```

## Files Affected

* `[MODIFY] internal/bff/bootstrap.go`
* `[MODIFY] internal/bff/bootstrap_test.go`
* `[MODIFY] internal/keeper/bootstrap.go`
* `[MODIFY] internal/keeper/bootstrap_test.go`
