# SPEC-FR-M6.5.2: LLM Binding CRUD APIs

| Field       | Value |
|-------------|-------|
| ID          | SPEC-FR-M6.5.2 |
| Status      | ACCEPTED |
| Milestone   | M6.5 |
| Component   | keeper |
| Depends On  | SPEC-FR-M3.4 |
| Supersedes  | SPEC-FR-M9.14 (DRAFT) |

## Context

LLM bindings configure the connection from an agent's brain to a specific LLM provider and model. The domain model (`internal/keeper/domain/model/llm_binding.go`) already exists with full validation. This spec defines the full CRUD REST endpoints, Redis caching, and the API contract. It also includes a **micro-fix**: the Agent aggregate's `LongTermMemoryConfig` is made fully optional (removing the mandatory `VectorDimension > 0` validation) since LTM is suspended in this milestone.

## Specification

### 1. CRUD Endpoints

- `GET  /api/v1/llm-bindings` — list all non-inactive bindings for the tenant (flat array, non-paginated).
- `POST /api/v1/llm-bindings` — create a new LLM binding.
- `GET  /api/v1/llm-bindings/{id}` — get by ID.
- `PUT  /api/v1/llm-bindings/{id}` — update mutable fields (Name, Description, DefaultModel, DefaultTemperature, DefaultMaxTokens, TimeoutSeconds, Status).
- `DELETE /api/v1/llm-bindings/{id}` — soft delete: sets `status = inactive`.

### 2. Secret Reference Handling

The `APIKeySecretRef` field stores a **Kubernetes Secret name reference** (e.g., `openai-api-key-secret`), never the raw key value. It is accepted on POST/PUT but **omitted from all GET responses** by using the `omitempty` JSON tag and clearing the field in the HTTP handler before returning the response.

### 3. Redis Caching

Cache individual binding records in Redis using key `{tenant_id}:llm-bindings:{id}` with TTL 300s. Invalidate on PUT and DELETE (soft delete).

### 4. Agent Association

No new association endpoint. The existing `Agent.Brain.LLMBindingID uuid.UUID` is set at agent creation. Validation at assignment time (SPEC-FR-M6.5.11) checks that the referenced binding has `status = active`.

### 5. LTM Micro-Fix

Remove mandatory `VectorDimension > 0` validation from `Agent.Validate()`. Add `omitempty` JSON tags to `LongTermMemoryConfig` fields. The Viper `bypass.ltm` flag in the agent runtime remains the operational gate.

### 6. Soft-Delete Name Reuse

To allow reusing binding names after soft-deletion, the unique constraint `unique_llm_bindings_tenant_name` is replaced by a partial unique index filtering out inactive bindings.

## Acceptance Criteria

1. `POST /api/v1/llm-bindings` creates a binding; `GET` returns it without the `api_key_secret_ref` value.
2. `DELETE` sets `status = inactive`; binding no longer appears in the list.
3. Creating an agent with an absent `long_term_memory` body field succeeds.
4. A second `GET` for the same binding is served from Redis cache.
5. An invalid `provider` value returns `422` with `{"error": "..."}`.

## Test Plan

- **Unit**: `LLMBinding.Validate()` for all invalid field combinations.
- **Unit**: `Agent.Validate()` succeeds with empty `LongTermMemoryConfig`.
- **Integration**: Gin handler tests in `gin.TestMode` with `httptest.NewRecorder()` for all five endpoints.
- **Integration**: Cache invalidation on PUT/DELETE.

## API Contract

```
POST   /api/v1/llm-bindings
       Body: { name, description?, provider, api_base_url, api_key_secret_ref,
               default_model, default_temperature, default_max_tokens, timeout_seconds }
       Response 201: LLMBinding (without api_key_secret_ref)

GET    /api/v1/llm-bindings          → []LLMBinding (without api_key_secret_ref)
GET    /api/v1/llm-bindings/{id}     → LLMBinding (without api_key_secret_ref)
PUT    /api/v1/llm-bindings/{id}     → updated LLMBinding (without api_key_secret_ref)
DELETE /api/v1/llm-bindings/{id}     → 204
```

## Files Affected

- `internal/keeper/domain/model/agent.go` [MODIFY] — LTM optional
- `internal/keeper/domain/model/llm_binding.go` [MODIFY] — add omitempty to APIKeySecretRef
- `internal/keeper/application/service/llm_binding_service.go` [MODIFY] — inject cache
- `internal/keeper/adapters/inbound/http/llm_binding_handler.go` [MODIFY] — empty secret ref, omit inactive in list
- `internal/keeper/adapters/outbound/postgres/llm_binding_repository.go` [MODIFY] — soft delete on Delete, exclude inactive in List
- `deploy/postgres/migrations/00003_llm_bindings_soft_delete_index.sql` [NEW] — drop unique constraint, add partial unique index
- `api/openapi/openapi.json` [MODIFY] — reflect API contract changes

