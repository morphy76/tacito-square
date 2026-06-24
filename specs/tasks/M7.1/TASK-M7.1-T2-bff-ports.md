# TASK-M7.1-T2: BFF Application Ports (`internal/bff/application/ports/`)

| Field       | Value                                            |
|-------------|--------------------------------------------------|
| Task ID     | TASK-M7.1-T2                                     |
| Spec        | SPEC-FR-M7.1                                     |
| Boundary    | BFF Application Ports — `internal/bff/application/ports/` |
| Status      | VERIFIED                                         |
| Depends On  | TASK-M7.1-T1                                     |

## Objective

Define all inbound and outbound port interfaces for the BFF application layer. These are the pure Go interfaces that decouple use cases from infrastructure. No implementation code exists in this task.

## Files

| File | Action |
|------|--------|
| `internal/bff/application/ports/inbound/session_usecase.go` | NEW |
| `internal/bff/application/ports/inbound/event_stream_usecase.go` | NEW |
| `internal/bff/application/ports/outbound/session_store.go` | NEW |
| `internal/bff/application/ports/outbound/oidc_provider.go` | NEW |
| `internal/bff/application/ports/outbound/keeper_client.go` | NEW |
| `internal/bff/application/ports/outbound/backend_event_client.go` | NEW |

## RED Phase

Write compile-time interface satisfaction tests (no logic, just `var _ Interface = (*ConcreteNil)(nil)` style assertions) in a test file for each port, ensuring that missing implementations cause compilation failures:

- `internal/bff/application/ports/outbound/session_store_test.go`:
  - Assert `SessionStore` interface requires `Save`, `Get`, `Delete`, and `DeleteByUserID` methods.
- `internal/bff/application/ports/outbound/oidc_provider_test.go`:
  - Assert `OIDCProvider` interface requires `ExchangeCode`, `RefreshToken`, `FetchUserInfo`, and `ValidateLogoutToken` methods.
- `internal/bff/application/ports/outbound/keeper_client_test.go`:
  - Assert `KeeperClient` interface is defined (at minimum as a stub with a `Ping(ctx) error` method, to be expanded by future feature specs as Keeper endpoints are consumed by the BFF).
- `internal/bff/application/ports/outbound/backend_event_client_test.go`:
  - Assert `BackendEventClient` interface requires `StreamEvents(ctx, tenantID) (<-chan []byte, error)` method.

Run `make test` — must fail (RED).

## GREEN Phase

1. Create `internal/bff/application/ports/outbound/session_store.go`:
   ```go
   type SessionStore interface {
       Save(ctx context.Context, session *model.Session, ttl time.Duration) error
       Get(ctx context.Context, sessionID string) (*model.Session, error)
       Delete(ctx context.Context, sessionID string) error
       DeleteByUserID(ctx context.Context, userID string) error
   }
   ```

2. Create `internal/bff/application/ports/outbound/oidc_provider.go`:
   ```go
   type TokenSet struct {
       AccessToken  string
       RefreshToken string
       IDToken      string
       ExpiresIn    time.Duration
   }
   type OIDCProvider interface {
       ExchangeCode(ctx context.Context, code, redirectURI string) (*TokenSet, error)
       RefreshToken(ctx context.Context, refreshToken string) (*TokenSet, error)
       FetchUserInfo(ctx context.Context, accessToken string) (*model.UserInfoPayload, error)
       ValidateLogoutToken(ctx context.Context, rawToken string) (sub string, sessionID string, err error)
   }
   ```

3. Create `internal/bff/application/ports/outbound/keeper_client.go`:
   ```go
   type KeeperClient interface {
       Ping(ctx context.Context) error
   }
   ```

4. Create `internal/bff/application/ports/outbound/backend_event_client.go`:
   ```go
   type BackendEventClient interface {
       StreamEvents(ctx context.Context, tenantID string) (<-chan []byte, error)
   }
   ```

5. Create `internal/bff/application/ports/inbound/session_usecase.go`:
   ```go
   type SessionUseCase interface {
       InitiateLogin(ctx context.Context) (authURL string, state string, err error)
       HandleCallback(ctx context.Context, code, state string) (*model.Session, error)
       RefreshSession(ctx context.Context, sessionID string) (*model.Session, error)
       Logout(ctx context.Context, sessionID string) error
       BackchannelLogout(ctx context.Context, rawLogoutToken string) error
       GetSession(ctx context.Context, sessionID string) (*model.Session, error)
   }
   ```

6. Create `internal/bff/application/ports/inbound/event_stream_usecase.go`:
   ```go
   type EventStreamUseCase interface {
       StreamEvents(ctx context.Context, tenantID string) (<-chan []byte, error)
   }
   ```

Run `make test` — tests must pass (GREEN).

## REFACTOR Phase

- Confirm all port interfaces live strictly in `application/ports/` and import **only** from `internal/bff/domain/model/` and standard library packages.
- Confirm no concrete types (structs, function bodies) exist in the port files.
- Verify `context.Context` is the first parameter on every interface method.
