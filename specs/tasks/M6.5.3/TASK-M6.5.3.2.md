# TASK-M6.5.3.2: Domain Layer — Agent Aggregate Modification & Prompt Resolution Service

| Field       | Value |
|-------------|-------|
| ID          | TASK-M6.5.3.2 |
| Status      | DRAFT |
| Spec        | SPEC-FR-M6.5.3 |
| Depends On  | none |

## Description

Modify the `Agent` aggregate domain model to support multiple attached prompts and prompt collections. Implement the core domain logic for resolving effective agent prompts (union-without-duplication, ordering, active status filtering).

## Work Items

1. **RED Phase**:
   - In `internal/keeper/domain/model/agent_test.go`, update test cases to verify the validation of `Agent` struct without `PromptTemplate` and with `Prompts` and `PromptCollections` fields.
   - Create unit tests under `internal/keeper/domain/service/prompt_resolution_test.go` verifying:
     - `TestResolvePrompts_UnionWithoutDuplication` (duplicate individually and via collections).
     - `TestResolvePrompts_CollectionOrder` (collection items appear first in definition order, then by collection attachment order, followed by directly attached items).
     - `TestResolvePrompts_SkipsDraftAndArchived` (only active prompts resolved, non-active skipped with warning log).

2. **GREEN Phase**:
   - Update `internal/keeper/domain/model/agent.go`:
     - Remove `PromptTemplate uuid.UUID` field from `Agent` struct.
     - Add `Prompts []uuid.UUID` and `PromptCollections []uuid.UUID` fields to the `Agent` struct.
     - Update validation in `Agent.Validate()` to remove references to the single `PromptTemplate` field.
   - Create `internal/keeper/domain/service/prompt_resolution.go`:
     - Implement `ResolveEffectivePrompts(ctx context.Context, agent *model.Agent, promptRepo outbound.PromptRepository) ([]*model.PromptTemplate, error)`.
     - Implement union-without-duplication resolution rules:
       1. For each `PromptCollections` entry (in agent attachment order), fetch collection templates and append `active` prompts if not already in the result set.
       2. For each directly attached prompt in `Prompts` (in attachment order), append if not already in the result set.
       3. Silently skip `draft` or `archived` templates, printing a WARN log entry: `"skipping non-active prompt {id} in agent resolution"`.

3. **REFACTOR Phase**:
   - Ensure clean interfaces and zero external infrastructure imports in the domain service.
   - Run tests: `go test ./internal/keeper/domain/...`.

## Acceptance Criteria

1. `Agent` struct supports `Prompts []uuid.UUID` and `PromptCollections []uuid.UUID`.
2. `ResolveEffectivePrompts` successfully resolves prompts in the correct order: collection-defined order first, then individual prompts.
3. Overlapping prompts are deduplicated by ID.
4. Non-active prompts (status is draft or archived) are excluded with a warning log, without returning an error.
5. All domain unit tests compile and pass GREEN.
