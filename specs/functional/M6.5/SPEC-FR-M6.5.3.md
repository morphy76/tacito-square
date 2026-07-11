# SPEC-FR-M6.5.3: Prompt & PromptCollection CRUD APIs + Agent Association

| Field       | Value                          |
|-------------|--------------------------------|
| ID          | SPEC-FR-M6.5.3                 |
| Status      | IMPLEMENTED                    |
| Milestone   | M6.5                           |
| Component   | keeper                         |
| Depends On  | SPEC-FR-M3.4                   |
| Supersedes  | SPEC-FR-M9.7 (DRAFT)           |

---

## Context

`PromptTemplate` and `PromptCollection` domain models already exist in `internal/keeper/domain/model/prompt.go`. This spec defines the full CRUD REST APIs for both, plus the agent-association endpoints.

The current `Agent` model carries a single `PromptTemplate uuid.UUID` field. This spec extends the model to support lists of directly attached prompts and attached collections, enabling richer system-prompt composition. Resolution semantics: union without duplication — collection items appear first (in collection-defined order, then by collection attachment order on the agent), followed by individually-attached prompts not already present in the set, deduplicated by ID.

---

## Specification

### 1. PromptTemplate CRUD

All endpoints are tenant-scoped. Tenant ID resolved and propagated via `context.Context`.

| Method | Path                    | Description                        |
|--------|-------------------------|------------------------------------|
| GET    | /api/v1/prompts         | List prompts for tenant            |
| POST   | /api/v1/prompts         | Create a new prompt template       |
| GET    | /api/v1/prompts/{id}    | Get a single prompt by ID          |
| PUT    | /api/v1/prompts/{id}    | Activate, archive, or update metadata |
| DELETE | /api/v1/prompts/{id}    | Soft-delete (set status=archived)  |

**Status lifecycle:** `draft` → `active` → `archived`. Activation is performed via `PUT` with `{"status": "active"}`. Content is **immutable** after creation; any content change creates a new prompt version record in `prompt_versions` (fields: `id`, `prompt_id`, `version_number`, `content_snapshot`, `created_at`).

### 2. PromptCollection CRUD

| Method | Path                                                           | Description                          |
|--------|----------------------------------------------------------------|--------------------------------------|
| GET    | /api/v1/prompt-collections                                     | List prompt collections for tenant   |
| POST   | /api/v1/prompt-collections                                     | Create a new collection              |
| GET    | /api/v1/prompt-collections/{id}                               | Get a single collection by ID        |
| PUT    | /api/v1/prompt-collections/{id}                               | Update collection metadata           |
| DELETE | /api/v1/prompt-collections/{id}                               | Soft-delete collection               |
| POST   | /api/v1/prompt-collections/{id}/prompts/{prompt_id}           | Add a prompt to the collection       |
| DELETE | /api/v1/prompt-collections/{id}/prompts/{prompt_id}           | Remove a prompt from the collection  |

Membership management endpoints return HTTP 200 with the updated collection object. Adding a prompt that is already a member returns HTTP 409.

### 3. Agent Association Endpoints

| Method | Path                                                              | Description                                       |
|--------|-------------------------------------------------------------------|---------------------------------------------------|
| POST   | /api/v1/agents/{id}/prompts/{prompt_id}                           | Attach individual prompt to agent                 |
| DELETE | /api/v1/agents/{id}/prompts/{prompt_id}                           | Detach individual prompt from agent               |
| POST   | /api/v1/agents/{id}/prompt-collections/{collection_id}            | Attach collection to agent                        |
| DELETE | /api/v1/agents/{id}/prompt-collections/{collection_id}            | Detach collection from agent                      |
| GET    | /api/v1/agents/{id}/prompts                                       | Returns the effective resolved prompt list        |

The `GET /api/v1/agents/{id}/prompts` endpoint returns the **resolved** prompt list using union-without-duplication semantics (see §5). Each item in the list includes: `id`, `name`, `content`, `status`, `source` (`"collection"` or `"individual"`), `collection_id` (if source is collection).

### 4. Agent Model Change

Replace the existing `PromptTemplate uuid.UUID` field on the `Agent` aggregate with:

```go
Prompts            []uuid.UUID `json:"prompts,omitempty"`
PromptCollections  []uuid.UUID `json:"prompt_collections,omitempty"`
```

The old `PromptTemplate` field is removed. A DB migration must backfill existing agent rows: if `prompt_template_id IS NOT NULL`, insert into `agent_prompts` table.

### 5. Resolution Semantics at Agent Spawn

When Keeper assembles the `PropagatedAgentConfig`, it resolves the effective prompt list as follows:

1. For each `PromptCollections` entry (in agent attachment order), iterate the collection's ordered member list. Include each `active` prompt not yet in the result set (deduplication by prompt ID).
2. For each directly attached prompt in `Prompts` (in attachment order), include it only if not already in the result set.
3. Prompts with status `draft` or `archived` are **silently skipped** with a `WARN` log entry: `"skipping non-active prompt {id} in agent resolution"`.
4. The ordered result is included in `PropagatedAgentConfig.Prompts` as `[]{ID, Name, Content}`.

### 6. Caching

- Cache the resolved prompt list per agent in Redis: key `agent-prompts:{tenantID}:{agentID}`, TTL `60s` (Viper key: `cache.agent_prompts_ttl`).
- Invalidate on: any agent-prompt or agent-prompt-collection attachment/detachment change; any prompt status change (`active`/`archived`); any collection membership change.

---

## Acceptance Criteria

1. **Prompt lifecycle**: `POST /api/v1/prompts` creates a prompt with `status=draft`. `PUT /api/v1/prompts/{id}` with `{"status": "active"}` transitions to `status=active`. `PUT /api/v1/prompts/{id}` with `{"status": "archived"}` transitions to `status=archived`.
2. **Agent association**: Attaching a prompt to an agent and calling `GET /api/v1/agents/{id}/prompts` returns it in the resolved list.
3. **Union-without-duplication**: Attaching the same prompt both individually and via a collection returns it only once in the resolved list, listed under `source="collection"` (collection takes precedence).
4. **Active-only filter**: A prompt with `status=draft` is not present in the `GET /api/v1/agents/{id}/prompts` resolved list, and no error is returned.
5. **Archived prompt excluded**: An archived prompt is excluded from resolution; no error is returned; a WARN log line is emitted.
6. **Content immutability**: Submitting a PUT with a changed `content` field creates a new version record; the original prompt record's content is unchanged.
7. **Cache invalidation**: After detaching a prompt, a subsequent `GET /api/v1/agents/{id}/prompts` reflects the removal (cache miss, fresh DB read).

---

## Test Plan

### Unit Tests

- `internal/keeper/domain/service/prompt_resolution_test.go`
  - `TestResolvePrompts_UnionWithoutDuplication` — collection + individual with overlap returns deduplicated list.
  - `TestResolvePrompts_CollectionOrder` — items from collection appear before individually-attached items.
  - `TestResolvePrompts_SkipsDraft` — draft prompt absent from result; no error.
  - `TestResolvePrompts_SkipsArchived` — archived prompt absent; WARN log emitted.
  - `TestResolvePrompts_MultipleCollections` — two collections with shared member; member appears once.

### Integration Tests (Gin TestMode + httptest)

- `internal/keeper/adapters/inbound/http/prompt_handler_test.go`
  - `TestCreatePrompt_DefaultsDraft` — verifies status=draft on creation.
  - `TestActivatePrompt` — verifies status transitions to active.
  - `TestAddPromptToCollection_Success` — verifies membership added.
  - `TestAddPromptToCollection_Duplicate_Returns409` — verifies 409 on duplicate add.
  - `TestAttachPromptToAgent` — verifies GET resolved list includes the prompt.
  - `TestDetachPromptFromAgent_CacheInvalidated` — mock Redis; verifies DEL on detach.
  - `TestGetAgentPrompts_UnionDeduplication` — full E2E union logic via handler.

---

## API Contract

### POST /api/v1/prompts

**Request Body:**
```json
{
  "name": "Customer Support Intro",
  "description": "Opening system prompt for customer support agents",
  "content": "You are a helpful customer support agent for Acme Corp..."
}
```

**Response 201:**
```json
{
  "id": "661f9500-e29b-41d4-a716-446655440001",
  "tenant_id": "tenant-abc",
  "name": "Customer Support Intro",
  "description": "Opening system prompt for customer support agents",
  "content": "You are a helpful customer support agent for Acme Corp...",
  "status": "draft",
  "version_number": 1,
  "created_at": "2026-07-10T08:00:00Z",
  "updated_at": "2026-07-10T08:00:00Z"
}
```

### PUT /api/v1/prompts/{id} — Activate

**Request Body:**
```json
{ "status": "active" }
```

**Response 200:** Updated prompt object with `status=active`.

### POST /api/v1/prompt-collections/{id}/prompts/{prompt_id}

**Response 200:** Updated collection object including membership list.
**Response 409:** `{"error": "prompt already member of collection"}`

### POST /api/v1/agents/{id}/prompts/{prompt_id}

**Response 200:** `{"message": "prompt attached to agent"}`
**Response 404:** `{"error": "agent not found"}` or `{"error": "prompt not found"}`

### GET /api/v1/agents/{id}/prompts

**Response 200:**
```json
{
  "agent_id": "agent-uuid",
  "resolved_prompts": [
    {
      "id": "prompt-uuid-1",
      "name": "Customer Support Intro",
      "content": "You are a helpful customer support agent...",
      "status": "active",
      "source": "collection",
      "collection_id": "collection-uuid-1"
    },
    {
      "id": "prompt-uuid-2",
      "name": "Escalation Protocol",
      "content": "When the customer is frustrated...",
      "status": "active",
      "source": "individual",
      "collection_id": null
    }
  ],
  "total": 2
}
```

### Error Response (all endpoints)

```json
{ "error": "descriptive error message" }
```

### OpenAPI Tag

All prompt operations carry the tag: `agent-config/prompts`
All prompt-collection operations carry the tag: `agent-config/prompt-collections`

---

## Files Affected

| File                                                                      | Action                                                               |
|---------------------------------------------------------------------------|----------------------------------------------------------------------|
| `internal/keeper/domain/model/prompt.go`                                  | EXISTING — no structural changes                                     |
| `internal/keeper/domain/model/agent.go`                                   | MODIFY — replace `PromptTemplate uuid.UUID` with `Prompts []uuid.UUID` and `PromptCollections []uuid.UUID` |
| `internal/keeper/domain/service/prompt_resolution.go`                     | NEW — union-without-duplication resolution logic                     |
| `internal/keeper/application/ports/inbound/prompt_service.go`             | NEW — inbound port interface                                         |
| `internal/keeper/application/ports/outbound/prompt_repository.go`         | NEW — outbound port interface                                        |
| `internal/keeper/application/service/prompt_service.go`                   | NEW — use case orchestration                                         |
| `internal/keeper/adapters/inbound/http/prompt_handler.go`                 | NEW — Gin HTTP handler                                               |
| `internal/keeper/adapters/outbound/postgres/prompt_repository.go`         | NEW — pgx adapter                                                    |
| `internal/keeper/adapters/outbound/redis/prompt_cache.go`                 | NEW — Redis cache adapter                                            |
| `db/migrations/<timestamp>_create_prompts.sql`                            | NEW — prompts, prompt_versions, prompt_collections, prompt_collection_members, agent_prompts, agent_prompt_collections tables; backfill from prompt_template_id |
