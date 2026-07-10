# TASK-M6.5.1.4: Keeper HTTP Inbound Adapter — Assignment Handlers

| Field       | Value |
|-------------|-------|
| ID          | TASK-M6.5.1.4 |
| Status      | IMPLEMENTED |
| Spec        | SPEC-FR-M6.5.1 |
| Depends On  | TASK-M6.5.1.2 |

## Description

Update the Gin HTTP inbound adapter for assignment endpoints in `internal/keeper/adapters/inbound/http/assignment_handlers.go`. Changes cover three areas:
- `Assign`: bind `role` from the JSON request body; forward it to the use-case; return `201 Created` with the assignment response body instead of `200 OK`.
- `Unassign`: no behavioral change needed; verify it returns `204 No Content` (already correct).
- `ListAssignments` (new): handle `GET /api/v1/communities/:community_id/agents` and return the assignment list.

## Work Items

1. **RED Phase**:
   - Update `internal/keeper/adapters/inbound/http/assignment_handlers_test.go`:
     - Test `Assign` returns `201 Created` with body `{ "agent_id": "...", "role": "...", "assigned_at": "..." }` on success.
     - Test `Assign` passes the `role` field from the request body to the use-case mock.
     - Test `Assign` returns `409 Conflict` when the use-case returns a hub-conflict domain error.
     - Test `Assign` returns `409 Conflict` when the use-case returns a topology/role mismatch domain error.
     - Test `Assign` returns `400 Bad Request` when the request body is missing `agent_id`.
     - Add test `ListAssignments` returns `200 OK` with array of assignment objects.
     - Add test `ListAssignments` returns `404 Not Found` when community does not exist.
   - All tests use `gin.SetMode(gin.TestMode)` and `httptest.NewRecorder()`.

2. **GREEN Phase**:
   - Define the request/response structs:
     ```go
     type assignRequest struct {
         AgentID uuid.UUID      `json:"agent_id" binding:"required"`
         Role    model.AgentRole `json:"role"`
     }
     type assignmentResponse struct {
         AgentID    uuid.UUID  `json:"agent_id"`
         Role       string     `json:"role"`
         AssignedAt time.Time  `json:"assigned_at"`
     }
     type assignmentListItem struct {
         AgentID    uuid.UUID  `json:"agent_id"`
         Role       string     `json:"role"`
         AssignedAt time.Time  `json:"assigned_at"`
         InformedAt *time.Time `json:"informed_at,omitempty"`
     }
     ```
   - Modify `Assign`:
     - Bind body with `c.ShouldBindJSON(&req)`.
     - Call `h.usecase.Assign(ctx, communityID, req.AgentID, req.Role)`.
     - Map hub-conflict and topology-mismatch domain errors to `409 Conflict`.
     - On success return `201 Created` with `assignmentResponse`.
   - Add `ListAssignments` handler:
     - Parse `:community_id` param.
     - Call `h.usecase.ListByCommunity(ctx, communityID)`.
     - Map not-found to `404`; return `200` with `[]assignmentListItem`.
   - Register new route in the existing `RegisterRoutes` function.

3. **REFACTOR Phase**:
   - Extract a shared `mapDomainErrorToHTTP` helper (or extend the existing error-string matching) to avoid duplicating `strings.Contains` logic across handlers.
   - Ensure OTel span names follow the existing naming convention (`http.<operation>`).

## Acceptance Criteria

1. All handler tests pass GREEN in `gin.TestMode` via `ServeHTTP`.
2. `POST` assignment endpoint returns `201 Created` with a structured JSON body.
3. `GET` assignments endpoint is registered and returns `200 OK` with the full assignment list.
4. `409 Conflict` is returned for hub-duplicate and topology-role mismatch errors.
5. No `strings` package-based error mapping is duplicated — a single helper is used.
