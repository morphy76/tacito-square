# TASK-M6.5.4.4: HTTP Handlers — New Endpoints + Hub Warning

| Field       | Value |
|-------------|-------|
| ID          | TASK-M6.5.4.4 |
| Status      | VERIFIED |
| Spec        | SPEC-FR-M6.5.4 |
| Depends On  | TASK-M6.5.4.3 |

## Description

Add the missing HTTP endpoints to the Gin handlers and enrich the community assignment response with hub skill warnings. All handlers follow the Gin test mode + httptest integration pattern established by `skill_handlers_test.go`.

## Work Items

### 1. `PATCH /api/v1/skills/{id}` — Status Transition

Add to `internal/keeper/adapters/inbound/http/skill_handlers.go`:

```go
// PatchSkillStatusRequest defines the PATCH payload.
type PatchSkillStatusRequest struct {
    Status model.SkillStatus `json:"status" binding:"required"`
}
```

Handler `PatchStatus(c *gin.Context)`:
- OTel span: `"http.patch_skill_status"`.
- Validate tenant, parse `:id` UUID, bind `PatchSkillStatusRequest`.
- Call `h.repo.PatchStatus(ctx, id, req.Status)`.
- Map `not found` → 404, `invalid status` → 400.
- Return 200 with the updated `Skill` object.

### 2. `POST /api/v1/skill-collections/{id}/skills/{skill_id}` — Add Skill to Collection

Add to `internal/keeper/adapters/inbound/http/skill_collection_handlers.go`:

Handler `AddSkillToCollection(c *gin.Context)`:
- Parse `:id` (collection) and `:skill_id` (skill) UUIDs.
- Call `h.repo.AddSkillToCollection(ctx, collectionID, skillID)` (via `SkillUseCase`).
- If error contains `"already member"` → return 409.
- On success: fetch updated collection via `h.repo.GetCollectionByID` and return 200 with collection object.

### 3. `DELETE /api/v1/skill-collections/{id}/skills/{skill_id}` — Remove Skill from Collection

Handler `RemoveSkillFromCollection(c *gin.Context)`:
- Parse both UUIDs.
- Call `h.repo.RemoveSkillFromCollection`.
- Map `not found` → 404, success → 200 with updated collection.

### 4. `POST /api/v1/agents/{id}/skill-collections/{collection_id}` — Attach Collection to Agent

Add to `internal/keeper/adapters/inbound/http/skill_handlers.go`:

Handler `AttachCollectionToAgent(c *gin.Context)`:
- OTel span: `"http.attach_collection_to_agent"`.
- Parse `:agent_id` and `:collection_id` UUIDs.
- Call `h.repo.AttachCollectionToAgent(ctx, agentID, collectionID)`.
- Map `not found` → 404, success → 200 `{"status":"attached"}`.

### 5. `DELETE /api/v1/agents/{id}/skill-collections/{collection_id}` — Detach Collection from Agent

Handler `DetachCollectionFromAgent(c *gin.Context)`:
- OTel span: `"http.detach_collection_from_agent"`.
- On success → 204 No Content.

### 6. `GET /api/v1/agents/{id}/skills` — Resolved Skill List

Handler `GetResolvedSkills(c *gin.Context)`:
- OTel span: `"http.get_resolved_agent_skills"`.
- Parse `:agent_id` UUID.
- Call `h.repo.ResolveAgentSkills(ctx, agentID)`.
- Return 200 with:
```go
type ResolvedSkillsResponse struct {
    AgentID       uuid.UUID              `json:"agent_id"`
    ResolvedSkills []*model.ResolvedSkill `json:"resolved_skills"`
    Total         int                    `json:"total"`
}
```

### 7. Hub Assignment Warning — `assignment_handlers.go`

Modify `Assign(c *gin.Context)` in `internal/keeper/adapters/inbound/http/assignment_handlers.go`:

- The `AssignmentHandler` must gain access to `AgentUseCase` (or the agent fetching capability). Inject it via constructor or use `SkillUseCase.ResolveAgentSkills` to check skill presence.
- After a successful `h.usecase.Assign(ctx, ...)`:
  1. Fetch the agent via `agentUseCase.GetByID(ctx, req.AgentID)`.
  2. If `req.Role == model.AgentRoleHub` and `len(agent.Skills) > 0 || len(agent.SkillCollections) > 0`, populate `warnings`.
- Extend the `assignmentResponse` struct:

```go
type assignmentResponse struct {
    AgentID    uuid.UUID  `json:"agent_id"`
    Role       string     `json:"role"`
    AssignedAt time.Time  `json:"assigned_at"`
    Warnings   []string   `json:"warnings,omitempty"`
}
```

### 8. Route Registration

Register all new routes in the appropriate `RegisterRoutes` function (or wherever routes are wired). Follow the existing registration convention for the `SkillHandler`:

```
PATCH  /api/v1/skills/:id
POST   /api/v1/skill-collections/:id/skills/:skill_id
DELETE /api/v1/skill-collections/:id/skills/:skill_id
POST   /api/v1/agents/:agent_id/skill-collections/:collection_id
DELETE /api/v1/agents/:agent_id/skill-collections/:collection_id
GET    /api/v1/agents/:agent_id/skills
```

### 9. Integration Tests

In `internal/keeper/adapters/inbound/http/skill_handlers_test.go`:

- `TestPatchSkillStatus_Suspend` — PATCH to `suspended`, verify 200 and `status=suspended`.
- `TestPatchSkillStatus_InvalidValue_Returns400` — PATCH with `"status":"unknown"` → 400.
- `TestAddSkillToCollection_Success` — POST membership, verify 200 + updated collection.
- `TestAddSkillToCollection_Duplicate_Returns409` — duplicate add → 409.
- `TestRemoveSkillFromCollection_Success` — DELETE membership → 200.
- `TestAttachCollectionToAgent_Success` — POST, verify 200 `{"status":"attached"}`.
- `TestDetachCollectionFromAgent_Success` — DELETE, verify 204.
- `TestGetResolvedSkills_ReturnsUnion` — agent with collection + individual skill → both in resolved list.
- `TestHubAssignment_WithSkills_WarningPresent` — assign hub role to agent with `Skills` → 201 with `warnings[]` non-empty.
- `TestHubAssignment_NoSkills_NoWarning` — hub with empty skills → 201, `warnings` absent or empty.

## Acceptance Criteria

1. All 10 integration tests pass.
2. `PATCH /api/v1/skills/{id}` returns 400 for unknown status values.
3. `POST /api/v1/skill-collections/{id}/skills/{skill_id}` returns 409 on duplicate.
4. Hub assignment response includes `warnings` only when skills are present.
5. All new routes registered with correct OTel span names and OpenAPI tags.
