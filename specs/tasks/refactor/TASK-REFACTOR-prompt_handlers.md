# TASK-REFACTOR-prompt_handlers: Refactor prompt_handlers.go

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-REFACTOR-prompt_handlers               |
| Status        | VERIFIED                                    |
| Target File   | [prompt_handlers.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/inbound/http/prompt_handlers.go)  |
| Baseline Tests| All existing tests MUST pass without changes |

## Description
Decompose the oversized `prompt_handlers.go` file (~584 LOC) by extracting the Prompt Collection handler endpoints, request payload structures, and related helpers into a separate file named `prompt_collection_handlers.go`. This simplifies the handlers layout, isolates the distinct prompt template and prompt collection concepts, and improves overall code maintainability without modifying any existing tests.

## Work Items
1. **Baseline Phase**:
   - Verify all existing tests pass (`make test`).
2. **Refactor Phase**:
   - [x] Create [prompt_collection_handlers.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/inbound/http/prompt_collection_handlers.go) in the `http` package.
   - [x] Move collection-specific structs (`CreateCollectionRequest`, `UpdateCollectionRequest`) to [prompt_collection_handlers.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/inbound/http/prompt_collection_handlers.go).
   - [x] Move collection-specific handler methods (`CreateCollection`, `GetCollectionByID`, `ListCollections`, `UpdateCollection`, `DeleteCollection`, `ResolveCollection`) to [prompt_collection_handlers.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/inbound/http/prompt_collection_handlers.go).
   - [x] Clean up imports in both files to keep only what's used.
3. **Verification Phase**:
   - Run the existing tests (`make test`) to ensure they remain 100% green.
   - Run `make lint` to confirm codebase styling compliance.

## Acceptance Criteria
1. No existing unit/integration/contract test is modified.
2. The target file `prompt_handlers.go` is cleaner and shorter.
3. The new file `prompt_collection_handlers.go` is created in the same package containing collection handlers.
4. Lint checks and tests pass cleanly.
