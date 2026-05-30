# TASK-M4.8-T7: Bootstrap & Keeper Wiring

| Field       | Value                                           |
|-------------|-------------------------------------------------|
| Task ID     | TASK-M4.8-T7                                    |
| Spec        | SPEC-FR-M4.8                                    |
| Boundary    | Bootstrap / Wiring — `internal/keeper/bootstrap.go` |
| Status      | IMPLEMENTED                                     |
| Depends On  | TASK-M4.8-T3, TASK-M4.8-T4, TASK-M4.8-T5      |


## Objective

Wire all new echo components into the Keeper's `bootstrap.go`: construct the `NATSCommunityBroadcaster`, `EchoServiceImpl`, and `EchoHandler`, and register the echo route on the `v1` group. Also update `openapi.json` to include the new endpoint.

## Files

| File | Action |
|------|--------|
| `internal/keeper/bootstrap.go` | MODIFY |
| `internal/keeper/openapi.json` | MODIFY |

## RED Phase

Extend `internal/keeper/bootstrap_test.go`:

- `TestNewServer_EchoRouteRegistered`: Call `NewServer(pool, nc, k8sConfig)` (or with a mock NATS conn). Use `httptest` to `GET` the route table or make a `POST /api/v1/communities/00000000-0000-0000-0000-000000000000/echo` request (expect `404 Not Found` for unknown community, not `404` for unknown route). Assert the route is registered.
- `TestNewServer_NilNATS_EchoReturns503`: Call `NewServer(pool, nil, k8sConfig)`. POST to the echo route. Assert `503 Service Unavailable` with `{"error": "NATS messaging is not available"}`.

Run `make test` — tests must fail (RED).

## GREEN Phase

**Modify `internal/keeper/bootstrap.go`**:

Locate the section where `lifecycleHandler` is constructed and routes are registered. Add after it:

```go
// Echo feature
natsBroadcaster := natsBroadcaster.NewNATSCommunityBroadcaster(nc, logger)
echoService := service.NewEchoService(communityRepo, agentRepo, crdCoord, natsBroadcaster, cfg)
echoHandler := http.NewEchoHandler(echoService)
echoHandler.RegisterRoutes(v1)
```

Import the new packages:
- `natsBroadcaster "github.com/morphy76/tacito-square/internal/keeper/adapters/outbound/nats"`

> **Note on `cfg`**: `NewEchoService` requires a `*viper.Viper` for timeout configuration. If the existing `NewServer` signature does not accept Viper, pass nil (service will use defaults) or extend the signature — prefer extending to keep configuration injectable.

**Modify `internal/keeper/openapi.json`**:

Add the new endpoint path to the OpenAPI document:

```json
"/api/v1/communities/{community_id}/echo": {
  "post": {
    "tags": ["community/echo"],
    "summary": "Fan out a message to all running agents in a community and collect decorated replies.",
    "operationId": "echoCommunity",
    "parameters": [
      {
        "name": "community_id",
        "in": "path",
        "required": true,
        "schema": { "type": "string", "format": "uuid" }
      }
    ],
    "requestBody": {
      "required": true,
      "content": {
        "application/json": {
          "schema": {
            "type": "object",
            "required": ["message"],
            "properties": {
              "message": { "type": "string", "maxLength": 1000 }
            }
          }
        }
      }
    },
    "responses": {
      "200": {
        "description": "Aggregated decorated replies from all running agents.",
        "content": {
          "application/json": {
            "schema": { "$ref": "#/components/schemas/CommunityEchoResponse" }
          }
        }
      },
      "400": { "description": "Empty or invalid message / invalid community_id." },
      "401": { "description": "Missing tenant context." },
      "404": { "description": "Community not found." },
      "503": { "description": "No running agents or NATS unavailable." }
    }
  }
}
```

Also add the component schemas `CommunityEchoResponse`, `AgentEchoResult` to `#/components/schemas`.

Also add `community/echo` to the top-level `tags` array with a description:
```json
{ "name": "community/echo", "description": "Community-scoped agent fanout messaging (echo)" }
```

Run `make test` and `make build-keeper` — must compile and pass (GREEN).

## REFACTOR Phase

- Run `make lint` and address any import cycle or unused variable warnings.
- Confirm the route appears in `GET /openapi.json` response at runtime (verify JSON is valid and the path is present).
- Confirm `GET /swagger/` (development mode) shows the `community/echo` tag with the new endpoint.
- Confirm that when `nc == nil`, `NATSCommunityBroadcaster.Available()` returns `false` and the service returns `ErrBroadcasterUnavailable` immediately, resulting in `503` — no nil pointer dereference.
- Run full `make test` — all existing tests GREEN, new bootstrap tests GREEN.

## Post-Task Verification (Full Manual Flow)

After T7 completes, run the full manual verification from SPEC-FR-M4.8:

1. `kubectl apply` two `TacitoAgent` CRs in the same community with `scaleToZeroEnabled: true`. Wait for idle scale-down (M4.4).
2. `curl -X POST http://keeper:8081/api/v1/communities/{id}/echo -H "Content-Type: application/json" -H "X-Tenant-ID: t1" -d '{"message":"hello world"}'`
3. Confirm response has `woke_community: true` and two result entries.
4. Confirm operator logs show scale-up triggered by NATS messages.
5. Confirm agent logs show `message = "hello world"` at `info` level.
6. Confirm `GET /openapi.json` includes the echo path with `community/echo` tag.
