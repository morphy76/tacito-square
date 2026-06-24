# SPEC-FR-M10.10: HTTP Cache Headers for BFF Static Resources

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M10.10                              |
| Status        | VERIFIED                                    |
| Milestone     | M10                                         |
| Component     | bff                                         |
| Depends On    | SPEC-FR-M7.7, SPEC-FR-M10.9                 |
| Supersedes    | none                                        |

## Context

The BFF serves static resources like index.html (welcome screen), secure/index.html (secure area entry), and favicon.ico. Currently, these assets are served without HTTP caching or conditional requests logic (except favicon, which has Cache-Control but lacks ETag/conditional check support).
To optimize performance, reduce bandwidth, and leverage caching infrastructure (browser cache, CDNs, proxies), we should implement strong caching headers (Cache-Control and ETags) for these resources.

## Specification

1. **HTML Entry Points (welcomeHTML and secureIndexHTML)**:
   - For paths `/ui`, `/ui/`, `/ui/index.html`, `/ui/secure`, `/ui/secure/`, and `/ui/secure/index.html`, the BFF MUST return:
     * `Cache-Control`: `public, max-age=0, must-revalidate`
     * `ETag`: A strong entity tag calculated using the SHA-256 hash of the specific HTML payload.
   - The SHA-256 hashes MUST be pre-computed at server startup to prevent CPU overhead.
   - The server MUST intercept incoming conditional requests containing the `If-None-Match` header matching the corresponding ETag and return a `304 Not Modified` status with an empty body.

2. **Favicon (favicon.ico)**:
   - For the path `/favicon.ico`, the BFF MUST return:
     * `Cache-Control`: `public, max-age=604800`
     * `ETag`: A strong entity tag calculated using the SHA-256 hash of the favicon payload.
   - The SHA-256 hash MUST be pre-computed at startup.
   - The server MUST intercept incoming conditional requests containing the `If-None-Match` header matching the favicon ETag and return a `304 Not Modified` status with an empty body.

## Acceptance Criteria

1. A request to public or secure HTML endpoints returns `Cache-Control: public, max-age=0, must-revalidate` and a valid double-quoted `ETag`.
2. A request to `/favicon.ico` returns `Cache-Control: public, max-age=604800` and a valid double-quoted `ETag`.
3. Subsequent requests with `If-None-Match` set to the resource's ETag return `304 Not Modified` with an empty body.
4. All existing tests in `internal/bff/bootstrap_test.go` pass.

## Test Plan

### Automated Tests
1. **Unit Tests:**
   - Add/update tests in `internal/bff/bootstrap_test.go` to assert Cache-Control, ETag headers, and conditional `If-None-Match` checks for public HTML, secure HTML, and favicon.
2. **Suite execution:**
   - Execute:
     ```bash
     make test
     ```

## Files Affected

- `internal/bff/bootstrap.go`
- `internal/bff/bootstrap_test.go`
