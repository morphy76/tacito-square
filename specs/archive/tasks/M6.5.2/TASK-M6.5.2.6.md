# TASK-M6.5.2.6: OpenAPI Contract Update

| Field       | Value |
|-------------|-------|
| ID          | TASK-M6.5.2.6 |
| Status      | IMPLEMENTED |
| Spec        | SPEC-FR-M6.5.2 |
| Depends On  | TASK-M6.5.2.5 |

## Description

Modify the centralized OpenAPI 3.x specification file `api/openapi/openapi.json` to reflect that the `api_key_secret_ref` field is omitted from all `GET` response schemas for LLM bindings.

## Work Items

1. **OpenAPI Schema Modification**:
   - Locate the `LLMBinding` schema inside `api/openapi/openapi.json` under `components.schemas`.
   - Remove `api_key_secret_ref` from the schema's `required` properties list.
   - Set the `api_key_secret_ref` property definition to not be required or add description text noting that it is omitted on output GET/PUT/POST operations.
   - Verify that `CreateLLMBindingRequest` and `UpdateLLMBindingRequest` schemas still mark `api_key_secret_ref` as required (since they receive inputs).

2. **Validation**:
   - Ensure the modified `openapi.json` conforms to the OpenAPI 3.x spec using lint tools if available or checking that the keeper project builds and tests run.

## Acceptance Criteria

1. `api/openapi/openapi.json` matches the implementation details (i.e. response definitions show that `api_key_secret_ref` is omitted or optional).
2. The project compiles and runs cleanly.
