# TASK-M6.5.3.3: Persistence Adapters — Agent and Prompt Repository Updates

| Field       | Value |
|-------------|-------|
| ID          | TASK-M6.5.3.3 |
| Status      | IMPLEMENTED |
| Spec        | SPEC-FR-M6.5.3 |
| Depends On  | TASK-M6.5.3.1 |

## Description

Modify outbound ports and implement database repository persistence for multiple attached prompts/collections on agents and prompt versions.

## Work Items

1. **RED Phase**:
   - In `internal/keeper/adapters/outbound/postgres/agent_repository_test.go`, write test cases for inserting, updating, and retrieving agents with attached `Prompts` and `PromptCollections`.
   - In `internal/keeper/adapters/outbound/postgres/prompt_repository_test.go`, write test cases verifying:
     - Creating prompt templates and versions.
     - Changing template content and verifying that a new version record is added to `prompt_versions` table and the main record's content column remains unchanged.
     - Adding/removing prompts from collections, and handling duplicate memberships (returning `409` conflict).

2. **GREEN Phase**:
   - **Outbound Ports**:
     - Update `PromptRepository` interface in `internal/keeper/application/ports/outbound/repositories.go`:
       - Add methods for versioning (`CreateVersion`, `GetLatestVersion`, etc.).
       - Add collection membership methods (`AddPromptToCollection`, `RemovePromptFromCollection`).
     - Update `AgentRepository` interface:
       - Add methods for prompt/collection association management if required.
   - **Agent Postgres Repository**:
     - Update `internal/keeper/adapters/outbound/postgres/agent_repository.go`:
       - Update queries in `Create`, `GetByID`, `GetByName`, `List`, and `Update` to read/write `Prompts` and `PromptCollections` via `agent_prompts` and `agent_prompt_collections` tables.
       - Implement helpers `loadPrompts`, `savePrompts`, `loadPromptCollections`, and `savePromptCollections` preserving position/ordering.
   - **Prompt Postgres Repository**:
     - Update `internal/keeper/adapters/outbound/postgres/prompt_repository.go`:
       - Implement version recording on `CreateTemplate` (initiate version 1).
       - In `UpdateTemplate`, check if prompt content is modified. If so, create a new record in `prompt_versions` (with incremented version number), but keep the `content` field in the main `prompt_templates` table unchanged.
       - Ensure `GetTemplateByID` and `ListTemplates` fetch/resolve the content from the latest version.
       - Implement collection membership add/remove operations. If adding a duplicate template, return a conflict error.

3. **REFACTOR Phase**:
   - Verify transaction boundary management across these repository queries.
   - Verify integration tests: `go test ./internal/keeper/adapters/outbound/postgres/...`.

## Acceptance Criteria

1. Postgres outbound repository tests pass.
2. Saving and loading an `Agent` aggregate successfully persists and restores its ordered `Prompts` and `PromptCollections`.
3. Updating prompt content creates a new row in `prompt_versions`, incrementing version number, and does not alter the original `prompt_templates.content` field.
4. Adding duplicate prompts to collections throws a 409 conflict error.
