# TASK-M6.5.2.5: HTTP Handler — API Key Secret Ref Masking & Errors

| Field       | Value |
|-------------|-------|
| ID          | TASK-M6.5.2.5 |
| Status      | TODO |
| Spec        | SPEC-FR-M6.5.2 |
| Depends On  | TASK-M6.5.2.4 |

## Description

Modify the `LLMBindingHandler` controllers to:
1. Mask the `api_key_secret_ref` in all API responses by clearing it (setting it to `""`) before serialization.
2. Standardize error responses to return `{"error": "..."}` on any validation or execution failure.

## Work Items

1. **RED Phase**:
   - Update `internal/keeper/adapters/inbound/http/llm_binding_handlers_test.go`:
     - Assert that GET, POST, and PUT JSON responses do not contain `api_key_secret_ref`.
     - Test that invalid input values return `400` or `422` with a `{"error": "..."}` JSON payload.

2. **GREEN Phase**:
   - Modify `internal/keeper/adapters/inbound/http/llm_binding_handlers.go`:
     - For single-record responses (`Create`, `GetByID`, `Update`), clear `APIKeySecretRef` (set it to `""`) on the returned model before serializing. (Since the `APIKeySecretRef` field has the `json:",omitempty"` tag, clearing it will omit it from the output).
     - For list responses (`List`), iterate through the bindings and clear `APIKeySecretRef` on each.
     - For any request parsing, domain validation, or service layer errors, return the proper HTTP status code and output a standard `c.JSON(status, gin.H{"error": err.Error()})` structure.

3. **REFACTOR Phase**:
   - Clean up handler logic and verify that OpenTelemetry trace instrumentation captures exceptions properly. Run handlers tests using `go test ./internal/keeper/adapters/inbound/http/...`.

## Acceptance Criteria

1. Inbound HTTP test suite passes cleanly.
2. All GET/POST/PUT response payloads contain no `api_key_secret_ref` property.
3. Every API failure response complies with the JSON format `{"error": "<description>"}`.
