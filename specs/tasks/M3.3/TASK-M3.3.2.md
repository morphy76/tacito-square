# TASK-M3.3.2: Skill Collections HTTP API Boundary

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M3.3.2                                 |
| Status        | IMPLEMENTED                                 |
| Spec          | SPEC-FR-M3.3                                |
| Depends On    | TASK-M3.3.1                                 |

## Description

Define the REST API request/response contracts, Gin routes, payload validation binding tags, and HTTP handler implementation for Skills (including agent-skill association endpoints). Follow a strict TDD lifecycle within this boundary.

## Work Items

1. **RED Phase**:
   - Create HTTP integration tests in `internal/keeper/adapters/http/skill_handlers_test.go` verifying:
     - `POST /api/v1/skills` creates a skill profile and returns 201.
     - `GET /api/v1/skills` retrieves active skills list.
     - `GET /api/v1/skills/{id}` retrieves single skill profile.
     - `PUT /api/v1/skills/{id}` updates attributes successfully.
     - `DELETE /api/v1/skills/{id}` deletes profile.
     - `POST /api/v1/agents/{agent_id}/skills/{skill_id}` attaches skill to agent and returns 200/201.
     - `DELETE /api/v1/agents/{agent_id}/skills/{skill_id}` detaches skill.
2. **GREEN Phase**:
   - Add input validation tags on HTTP request models.
   - Implement controller handlers in `internal/keeper/adapters/http/skill_handlers.go`.
   - Setup route mappings in `internal/keeper/bootstrap.go`.
3. **REFACTOR Phase**:
   - Refactor controller mapping checks to cleanly return standard JSON error structures on invalid IDs.
   - Decouple agent template relational models to keep bootstrap routing config DRY.

## Acceptance Criteria

1. `skill_handlers_test.go` HTTP controller integration tests pass successfully with race detector enabled.
2. Gin endpoints successfully validate fields, manage attachments/detachments via repository ports, and return standard JSON formats.
