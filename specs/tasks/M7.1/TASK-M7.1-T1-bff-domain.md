# TASK-M7.1-T1: BFF Domain Layer (`internal/bff/domain/`)

| Field       | Value                                    |
|-------------|------------------------------------------|
| Task ID     | TASK-M7.1-T1                             |
| Spec        | SPEC-FR-M7.1                             |
| Boundary    | BFF Domain — `internal/bff/domain/`      |
| Status      | IMPLEMENTED                              |
| Depends On  | —                                        |

## Objective

Define the pure domain layer of the BFF: the `Session` aggregate, view-model value objects, and domain-level errors. This layer has zero infrastructure imports and is the vocabulary shared across ports and services.

## Files

| File | Action |
|------|--------|
| `internal/bff/domain/model/session.go` | NEW |
| `internal/bff/domain/model/viewmodel.go` | NEW |
| `internal/bff/domain/model/errors.go` | NEW |
| `internal/bff/domain/model/session_test.go` | NEW |
| `internal/bff/domain/model/viewmodel_test.go` | NEW |

## RED Phase

Write unit tests in `session_test.go` and `viewmodel_test.go`:

- `TestSession_NewSession`: Assert that `NewSession(userID, tenantID, accessToken, refreshToken, userInfo)` generates a cryptographically random Session ID (UUID v4), sets `CreatedAt`, and marks `IsExpired()` as `false` on a fresh session.
- `TestSession_IsExpired`: Construct a session with a backdated `AccessTokenExpiresAt` and assert `IsExpired()` returns `true`.
- `TestSession_Invalidate`: Call `session.Invalidate()` and assert the session's internal state is marked as invalidated (e.g., all token fields zeroed/cleared).
- `TestUserInfoPayload_JSON`: Assert that `UserInfoPayload` correctly marshals/unmarshals `sub`, `email`, `tenantid`, and `subscriptionid` JSON fields.
- `TestViewModelErrors`: Assert that all sentinel domain errors (`ErrSessionNotFound`, `ErrSessionExpired`, `ErrSessionInvalidated`) implement the `error` interface and return distinct messages.

Run `make test` — must fail because the package doesn't exist (RED).

## GREEN Phase

1. Create `internal/bff/domain/model/session.go`:
   - Define `Session` struct with fields: `ID` (string), `UserID` (string), `TenantID` (string), `AccessToken` (string), `RefreshToken` (string), `UserInfo` (`UserInfoPayload`), `AccessTokenExpiresAt` (`time.Time`), `CreatedAt` (`time.Time`), `Invalidated` (bool).
   - Implement `NewSession(...) (*Session, error)` — generates UUID v4 for `ID` using `crypto/rand`, sets `CreatedAt` to `time.Now().UTC()`.
   - Implement `IsExpired() bool` — returns `time.Now().UTC().After(s.AccessTokenExpiresAt)`.
   - Implement `Invalidate()` — zeroes all token fields and sets `Invalidated = true`.

2. Create `internal/bff/domain/model/viewmodel.go`:
   - Define `UserInfoPayload` struct with JSON tags: `Sub string \`json:"sub"\``, `Email string \`json:"email"\``, `TenantID string \`json:"tenantid"\``, `SubscriptionID string \`json:"subscriptionid"\``.
   - Define placeholder view-model structs for the configurator and auditor namespaces (e.g., `ConfiguratorViewModel`, `AuditorViewModel`) as empty named types — to be expanded by subsequent feature specs.

3. Create `internal/bff/domain/model/errors.go`:
   - Declare sentinel errors: `ErrSessionNotFound`, `ErrSessionExpired`, `ErrSessionInvalidated` using `errors.New`.

Run `make test` — tests must pass (GREEN).

## REFACTOR Phase

- Confirm the domain package has **zero imports** from `application/` or `adapters/` packages.
- Verify all token fields in `Session` are tagged `json:"-"` to prevent accidental serialization.
- Confirm `NewSession` uses `crypto/rand` (not `math/rand`) for the session ID.
