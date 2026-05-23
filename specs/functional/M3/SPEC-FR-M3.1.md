# SPEC-FR-M3.1: LLM Provider Bindings & CRUD API

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M3.1                                |
| Status        | ACCEPTED                                    |
| Milestone     | M3                                          |
| Component     | keeper                                      |
| Depends On    | SPEC-FR-M2.3                                |
| Supersedes    | none                                        |

## Context

Agents require an integrated Large Language Model ("brain") to perform reasoning and processing. Rather than hardcoding LLM configuration parameters inside each agent template, keeper manages a catalog of reusable **LLM Provider Bindings**. These bindings encapsulate the API endpoints, models, credentials, and parameters needed to connect to standard OpenAI-compatible REST services, providing modularity and separation of concerns.

## Specification

1. The system MUST define an `LLMBinding` aggregate in the keeper domain layer representing an LLM provider connection configuration with fields:
   - `id`: UUID (Primary Key)
   - `name`: String (Unique, required - e.g., `openai-gpt4o`, `local-llama3`)
   - `description`: String (Optional)
   - `provider`: String (Enum: `openai`, `anthropic`, `groq`, `ollama`, `custom`)
   - `api_base_url`: String (Required, API gateway endpoint)
   - `api_key_secret_ref`: String (Name of the K8s Secret containing the API key)
   - `default_model`: String (Required, default model name)
   - `default_temperature`: Float (Optional, default 0.7)
   - `default_max_tokens`: Integer (Optional, default 2048)
   - `timeout_seconds`: Integer (Optional, default 30)
   - `status`: Enum (`active`, `suspended`, `inactive`)
   - `created_at`: Timestamp
   - `updated_at`: Timestamp
2. The keeper MUST expose CRUD REST endpoints to manage LLM provider bindings:
   - `POST /api/v1/llm-bindings`: Create a new LLM provider binding.
   - `GET /api/v1/llm-bindings`: List all LLM provider bindings.
   - `GET /api/v1/llm-bindings/{id}`: Retrieve a specific LLM provider binding.
   - `PUT /api/v1/llm-bindings/{id}`: Update an LLM provider binding.
   - `DELETE /api/v1/llm-bindings/{id}`: Delete an LLM provider binding.
3. The domain layer MUST NOT import adapter or application packages (per `SPEC-NFR-HEXAGONAL`).
4. Input validation MUST use Gin binding tags (per `SPEC-NFR-HTTP`).
5. The system MUST support securing credentials using reference names pointing to Kubernetes secrets (`api_key_secret_ref`) rather than persisting clear-text credentials.

## Acceptance Criteria

1. **Domain Model**:
   - `LLMBinding` struct defines all required properties and enforces uniqueness on the binding name.
2. **API Endpoint Integration**:
   - Creating, reading, updating, and deleting LLM bindings via the REST API works correctly with schema validation.
   - Invalid field values (e.g. invalid provider type, negative timeout) are rejected with a 400 Bad Request standard JSON error.
3. **Hexagonal Boundaries**:
   - Domain layer remains free of external dependencies.

## Test Plan

### Automated Tests
1. **Unit Tests**:
   - Verify input validation constraints on Gin structures.
   - Test LLM Binding domain model logic.
2. **Integration Tests**:
   - Verify CRUD REST API endpoints mock-tested via Gin engine.

## Files Affected

- `internal/keeper/domain/llm_binding.go` [NEW] — Defines the `LLMBinding` model.
- `internal/keeper/ports/repositories.go` [MODIFY] — Declares repository interface for LLM bindings.
- `internal/keeper/adapters/http/llm_binding_handlers.go` [NEW] — Implements API controllers for LLM bindings.
- `internal/keeper/bootstrap.go` [MODIFY] — Binds LLM binding routes onto the Gin engine.
