# SPEC-FR-M3.3: Skill Collections & CRUD API

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M3.3                                |
| Status        | ACCEPTED                                    |
| Milestone     | M3                                          |
| Component     | keeper                                      |
| Depends On    | SPEC-FR-M2.3, SPEC-FR-M3.2                  |
| Supersedes    | none                                        |

## Context

Skills represent an agent's specialization or capabilities. Rather than exposing all tools from every connected MCP server unconditionally, keeper manages **Skill Collections**. A Skill Collection groups specific tool packages and associates them with one or more MCP server configurations, with options to whitelist/blacklist specific tools. Agents are then assigned these skill collections, which are resolved at deploy-time, providing fine-grained access control and keeping the LLM context size clean.

## Specification

1. The system MUST define a `Skill` aggregate in the keeper domain layer representing a specialized capacity:
   - `id`: UUID (Primary Key)
   - `name`: String (Unique, required)
   - `description`: String (Optional)
   - `mcp_servers`: List of UUIDs referencing `MCPServer` configurations
   - `allowed_tools`: List of Strings (Optional whitelist of specific tool names; if empty, all tools are exposed)
   - `denied_tools`: List of Strings (Optional blacklist of specific tool names to block)
   - `status`: Enum (`active`, `suspended`, `inactive`)
   - `created_at`: Timestamp
   - `updated_at`: Timestamp
2. The keeper MUST expose CRUD REST endpoints to manage skills:
   - `POST /api/v1/skills`: Create a new skill profile.
   - `GET /api/v1/skills`: List all skills.
   - `GET /api/v1/skills/{id}`: Retrieve a specific skill profile.
   - `PUT /api/v1/skills/{id}`: Update a skill profile.
   - `DELETE /api/v1/skills/{id}`: Delete a skill profile.
3. The keeper MUST expose sub-endpoints to attach/detach skills to/from agents:
   - `POST /api/v1/agents/{agent_id}/skills/{skill_id}`: Associate a skill with an agent.
   - `DELETE /api/v1/agents/{agent_id}/skills/{skill_id}`: Remove a skill association from an agent.
4. The domain layer MUST NOT import adapter or application packages (per `SPEC-NFR-HEXAGONAL`).
5. Input validation MUST use Gin binding tags (per `SPEC-NFR-HTTP`).
6. At deployment resolution, keeper MUST aggregate all skills associated with an agent, resolve the underlying MCP server configurations, filter tools based on whitelists/blacklists, and inject the final authorized list into the CRD spec.

## Acceptance Criteria

1. **Domain Model**:
   - `Skill` aggregate cleanly maps whitelists and blacklists and references valid MCP servers.
2. **API Endpoint Integration**:
   - Creating, reading, updating, and deleting skills works as expected.
   - Validates that associated `mcp_servers` exist when creating/updating a skill (optional or checked during transaction).
   - Associating skills to agents successfully updates the agent's template configuration in the persistent layer.
3. **Hexagonal Boundaries**:
   - No framework leakage in the domain package.

## Test Plan

### Automated Tests
1. **Unit Tests**:
   - Test whitelisting and blacklisting resolution logic.
2. **Integration Tests**:
   - Test REST API controller for Skill creation and attachment to an Agent template via simulated Gin requests.

## Files Affected

- `internal/keeper/domain/skill.go` [NEW] — Defines the `Skill` model.
- `internal/keeper/ports/repositories.go` [MODIFY] — Declares repository ports for skills.
- `internal/keeper/adapters/http/skill_handlers.go` [NEW] — Implements API controllers for skills.
- `internal/keeper/bootstrap.go` [MODIFY] — Binds skill routes onto the Gin engine.
