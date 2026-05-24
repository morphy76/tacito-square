# TASK-M3.8.8: Secure Tenant ID Boundary Validation & Error Propagation Middleware

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M3.8.8                                 |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M3.8                                |
| Depends On    | none                                        |

## Description

Design and implement complete tenant verification checks within the Keeper HTTP handler middleware layer in compliance with `cloud-first.md`. If the resolved tenant context header (`X-Tenant-ID`) is completely missing, empty, or malformed, the middleware must immediately abort execution and return a standardized JSON error envelope carrying a `401 Unauthorized` status to ensure secure tenant isolation boundaries.

## Boundary & Target Functions

- **Package**: `internal/keeper/adapters/inbound/http`
- **File**: `internal/keeper/adapters/inbound/http/middleware.go`
- **Target Functions**:
  - `TenantResolutionMiddleware(resolver TenantResolver) gin.HandlerFunc`

## Work Items

1. **RED Phase**:
   - Write middleware unit tests in `internal/keeper/adapters/inbound/http/middleware_test.go` to assert:
     - Requests containing valid `X-Tenant-ID` headers are propagated successfully without errors.
     - Requests missing `X-Tenant-ID` headers are immediately intercepted and aborted with a `401 Unauthorized` HTTP status.
     - Requests containing empty or malformed spaces as tenant IDs are similarly rejected with `401 Unauthorized`.
     - Standard JSON response format matches `{ "error": "Clear and descriptive tenant resolution error message" }`.

2. **GREEN Phase**:
   - Update `TenantResolutionMiddleware` inside `internal/keeper/adapters/inbound/http/middleware.go`.
   - Ensure the resolved context checks for empty, whitespace-only, or missing values.
   - On error or validation failure, invoke `c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})` immediately.

3. **REFACTOR Phase**:
   - Streamline header parsing logic inside `HeaderTenantResolver` to ensure robust checking of multi-value arrays or trailing whitespaces.
   - Confirm unified JSON error structures comply perfectly with `nonfunctional.md` HTTP standards.

## Acceptance Criteria

1. Requests missing active tenant identifications fail gracefully with standard JSON payloads carrying a `401 Unauthorized` status.
2. Verified tenant contexts are seamlessly injected into Go request context objects and propagated downstream.
3. No request processing escapes tenant isolation boundary enforcement.
