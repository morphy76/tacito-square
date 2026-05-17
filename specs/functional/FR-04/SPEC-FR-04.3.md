# SPEC-FR-04.3: Long-Term Memory (Qdrant Adapter)

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-FR-04.3                       |
| Status        | IMPLEMENTED                        |
| Milestone     | M2                                 |
| FR/NFR Ref    | FR-04.3                            |
| Component     | agent                              |
| Depends On    | SPEC-FR-01.2                       |

## Context

Agents need persistent, searchable memory that survives thread boundaries. Qdrant provides vector similarity search for semantic recall of past interactions and learned facts.

## Specification

1. `LongTermMemory` outbound port (in `ports.go`) defines:
   - `Store(ctx, entry)` — persists a `MemoryEntry` with its embedding vector
   - `Search(ctx, agentID, embedding, topK)` — returns semantically similar entries
2. `QdrantClient` interface abstracts Qdrant operations:
   - `Upsert(ctx, collection, id, vector, payload)` — stores/updates a point
   - `Search(ctx, collection, vector, limit, filter)` — vector similarity search
3. `LTMAdapter` MUST:
   a. Reject entries with empty embeddings (return error)
   b. Store entries as Qdrant points with payload: `agent_id`, `content`, `kind`, `created_at`
   c. Optionally include `thread_id` in payload when present
   d. Filter search results by `agent_id` to enforce agent isolation
   e. Respect `topK` limit on search results
4. Collection name is configurable at adapter construction.

## Acceptance Criteria

1. Store succeeds and persists vector + metadata payload ✅
2. Store preserves agent_id, kind, created_at in payload ✅
3. Search returns matching results ✅
4. Search filters by agent_id (agent isolation) ✅
5. Search respects topK limit ✅
6. Search on empty collection returns empty slice ✅
7. Store with empty embedding returns error ✅

## Files

- `internal/agent/ports/outbound/ports.go` ✅ (LongTermMemory interface)
- `internal/agent/adapters/outbound/qdrant/ltm_adapter.go` ✅ IMPLEMENTED
- `internal/agent/adapters/outbound/qdrant/ltm_adapter_test.go` ✅ 7 tests
