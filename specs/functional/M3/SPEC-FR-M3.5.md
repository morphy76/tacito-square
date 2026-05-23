# SPEC-FR-M3.5: Agent Domain Model & CRUD API

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M3.5                                |
| Status        | IMPLEMENTED                                 |
| Milestone     | M3                                          |
| Component     | keeper                                      |
| Depends On    | SPEC-FR-M2.3, SPEC-FR-M3.1, SPEC-FR-M3.2, SPEC-FR-M3.3, SPEC-FR-M3.4 |
| Supersedes    | none                                        |

## Context

### Definition of an Agent
An agent is a computational unit that receives and emits messages into a mailbox for asynchronous processing. The adapter used for incoming and outgoing ports is **NATS**, which integrates functional messaging, observability, and troubleshooting capabilities.

The asynchronous processing within an active agent leverages several core characteristics:
- **Brain**: An integrated Large Language Model (LLM) utilizing OpenAI's de facto standard REST API signatures.
- **Multithreaded Engagement**: Supports multi-topic communication, enabling the same agent to "talk" and maintain contexts about multiple topics concurrently.
- **Short-term Memory**: Fast-access, ephemeral state tracking implemented via a **Redis** adapter.
- **Long-term Memory**: Persistent semantic storage of main concepts and previous conversation threads, realized using a **Qdrant** adapter.
- **Specialization (Skills)**: Functional modules or capabilities that the agent can invoke to perform specific technical tasks.
- **Custom Behavior (Prompts)**: Customizable behavior instructions defined through prompts.
- **Acting (MCP Clients)**: Invocation of tools and actions in external environments via Model Context Protocol (MCP) clients.

### Keeper's Role & Lifecycle
The **keeper** service manages the "templates" of agents and their supporting entities.
Deploying an agent means converting its template definition (stored in keeper) into an actually running unit, instantiated at runtime as a container managed by a custom resource definition (CRD) in the cluster environment.

## Specification

1. The system MUST define an `Agent` aggregate in the keeper domain layer representing the template of an agent. It MUST contain the following fields:
   - `id`: UUID (Primary Key)
   - `name`: String (Unique, required)
   - `description`: String (Optional)
   - `brain`: LLM configuration (model, temperature, max tokens, endpoint, credentials)
   - `short_term_memory`: Redis adapter configuration (e.g. key namespaces, TTL)
   - `long_term_memory`: Qdrant adapter configuration (e.g. collection name, vector dimension)
   - `skills`: List of associated Skill identifiers
   - `prompt_template`: Reference to custom system prompt or instructions
   - `mcp_clients`: List of MCP server connections/configurations for tools and actions
   - `status`: Enum (`defined`, `assigned`, `active`, `terminated`)
   - `created_at`: Timestamp
   - `updated_at`: Timestamp
2. The keeper MUST expose standard CRUD REST endpoints to manage agent templates:
   - `POST /api/v1/agents`: Create a new agent template definition.
   - `GET /api/v1/agents`: List all agent templates (with optional pagination/filters).
   - `GET /api/v1/agents/{id}`: Retrieve a specific agent template.
   - `PUT /api/v1/agents/{id}`: Update an existing agent template.
   - `DELETE /api/v1/agents/{id}`: Delete an agent template (with soft delete or dependency checks).
3. The domain layer MUST NOT import adapter or application packages (per `SPEC-NFR-HEXAGONAL`).
4. Input validation MUST use Gin binding tags (per `SPEC-NFR-HTTP`).
5. All responses MUST use the standard JSON error format on failure.
6. The deployment mapping MUST translate these agent templates into Kubernetes custom resources (CRDs) representing the active containerized agents.

## Acceptance Criteria

1. **Domain Model Integrity**:
   - The `Agent` aggregate in `internal/keeper/domain` includes all specified fields (brain, memories, skills, prompts, MCP config, status).
   - The model does not import any adapter packages.
2. **CRUD REST API**:
   - `POST /api/v1/agents` successfully validates input using Gin tags and persists the new agent template to the database.
   - `GET /api/v1/agents` and `GET /api/v1/agents/{id}` retrieve saved definitions correctly.
   - `PUT /api/v1/agents/{id}` allows modification of agent parameters (e.g., brain, prompt, skills) and validates fields properly.
   - `DELETE /api/v1/agents/{id}` correctly marks an agent template as deleted (or performs standard validation before removal).
3. **Error Handling**:
   - Failed validation or invalid resource IDs return a standard JSON error response containing error details and appropriate HTTP status codes (e.g., 400 Bad Request, 404 Not Found).

## Test Plan

### Automated Tests
1. **Unit Tests**:
   - Implement unit tests for the `Agent` domain model validation and template instantiation logic.
   - Implement unit tests for the domain service using mocked ports.
2. **Integration / HTTP Controller Tests**:
   - Set up test suites in `internal/keeper/adapters/http/controller_test.go` (or similar) mocking the repository/use cases.
   - Verify that Gin handlers respond correctly for valid and invalid payloads under all CRUD operations.
3. **Database Integration Tests**:
   - Verify persistence and migration behavior in GORM (once PostgreSQL integration is configured).

## Files Affected

- `internal/keeper/domain/agent.go` [NEW] — Defines the `Agent` aggregate and domain rules.
- `internal/keeper/ports/repositories.go` [MODIFY] — Declares the interface for agent template persistence.
- `internal/keeper/ports/handlers.go` [NEW] — Declares the handler signatures for the HTTP controller.
- `internal/keeper/adapters/http/handlers.go` [NEW] — Implements HTTP handlers using Gin for CRUD operations.
- `internal/keeper/adapters/postgres/repository.go` [NEW] — Implements database operations for agent persistence.
- `internal/keeper/bootstrap.go` [MODIFY] — Configures and binds the new agent routes onto the Gin engine.
