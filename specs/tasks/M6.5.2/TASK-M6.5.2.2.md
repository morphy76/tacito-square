# TASK-M6.5.2.2: Keeper Domain Models — LTM Optionality & Secret JSON Tag

| Field       | Value |
|-------------|-------|
| ID          | TASK-M6.5.2.2 |
| Status      | IMPLEMENTED |
| Spec        | SPEC-FR-M6.5.2 |
| Depends On  | none |

## Description

Modify the domain validations and struct definitions in `internal/keeper/domain/model/` to align with the specification rules:
1. Make `LongTermMemoryConfig` optional in `Agent.Validate()` (LTM is bypassed in this milestone). Add `omitempty` JSON tag to the subfields of `LongTermMemoryConfig`.
2. Add the `omitempty` tag to `APIKeySecretRef` in the `LLMBinding` struct to allow omitting it from serialization.

## Work Items

1. **RED Phase**:
   - Update `internal/keeper/domain/model/agent_test.go`:
     - Modify the test `"Invalid long term memory vector dimension"` or write a new test confirming that calling `Validate()` on an `Agent` template with an empty `LongTermMemory` configuration succeeds without returning an error.
   - Update `internal/keeper/domain/model/llm_binding_test.go` to ensure validation still works cleanly with `APIKeySecretRef` modifications.

2. **GREEN Phase**:
   - Modify `internal/keeper/domain/model/agent.go`:
     - Change the JSON tags of `LongTermMemoryConfig` fields to support `omitempty`.
     - In `Agent.Validate()`, remove the vector dimension constraint that requires `VectorDimension > 0`.
   - Modify `internal/keeper/domain/model/llm_binding.go`:
     - Update the `APIKeySecretRef` struct tag to:
       ```go
       APIKeySecretRef string `json:"api_key_secret_ref,omitempty"`
       ```

3. **REFACTOR Phase**:
   - Run the domain tests using `go test ./internal/keeper/domain/...` to ensure all tests pass GREEN.

## Acceptance Criteria

1. `go test ./internal/keeper/domain/model/...` passes successfully.
2. `Agent.Validate()` allows zero vector dimension for `LongTermMemoryConfig`.
3. `LLMBinding` struct definition includes `json:"api_key_secret_ref,omitempty"` for the `APIKeySecretRef` field.
