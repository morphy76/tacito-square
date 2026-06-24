# TASK-REFACTOR-skill_handlers: Refactor skill_handlers.go

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-REFACTOR-skill_handlers                |
| Status        | VERIFIED                                    |
| Target File   | [skill_handlers.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/inbound/http/skill_handlers.go)  |
| Baseline Tests| All existing tests MUST pass without changes |

## Description
Split `SkillCollection` handlers and their associated payload structs out of `skill_handlers.go` into a new file `skill_collection_handlers.go` within the same package. Introduce a helper `parseUUID` on `SkillHandler` to eliminate duplicate parsing blocks.

## Work Items
1. **Baseline Phase**:
   - [x] Verify all existing tests pass.
2. **Refactor Phase**:
   - [x] Create a new file [skill_collection_handlers.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/inbound/http/skill_collection_handlers.go) and move all `SkillCollection` handlers and their request structs to it.
   - [x] Add the internal helper `parseUUID` to `SkillHandler` inside `skill_handlers.go` to handle UUID parsing and HTTP bad request responses.
   - [x] Refactor all handlers in both files to use `parseUUID` helper.
3. **Verification Phase**:
   - [x] Run existing tests to ensure they are 100% green.
   - [x] Run `make lint` to confirm codebase styling compliance.

## Acceptance Criteria
1. No existing unit/integration/contract test is modified.
2. [skill_handlers.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/inbound/http/skill_handlers.go) has its LOC reduced from 671 lines to under 400 lines.
3. All existing HTTP handler tests remain fully green.
4. Lint checks pass cleanly.
