# SDD Workflow: Spec Creation (Interactive Drafting)

This workflow defines the step-by-step interactive process for creating a new **Functional Specification (FR)** inside the Tacito Square repository, fully aligned with the [Project Constitution](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/constitution.md) and tracked via the master [Specs Index](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/INDEX.md).

---

## Step 1: The Interactive Interview Phase
Governed by Principle P1 (**Spec-Driven Development**) of the [Project Constitution](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/constitution.md), the agent MUST NOT draft or write a specification in isolation. You must conduct a structured, interactive interview with the User to gather precise details. Ask questions one at a time or in brief groups, focusing on:

1.  **Scope & Context**: What is the core business problem being solved? What is the background context?
2.  **Bounded Context & Components**: Which architectural components are affected (`keeper`, `agent`, `operator`, `bff`, `shared`, `deploy`)?
3.  **RFC 2119 Specifications**: What are the strict `MUST`, `SHOULD`, and `MAY` requirements?
4.  **Acceptance Criteria**: What are the concrete, verifiable conditions for success?
5.  **Test Plan & API Contracts**: How will we verify this? If it exposes HTTP/REST or messaging, what are the payload structures and status codes?
6.  **Milestones & Dependencies**: Which milestone does this target? Does it depend on other specifications?

---

## Step 2: Spec File Creation & Indexing
Once the User has provided sufficient details, create the specification file using the exact template below. Every specification file MUST follow the structural standard defined in Section 4 (**Spec Document Format**) of the [Project Constitution](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/constitution.md):

1.  **Path**: `specs/functional/M<Milestone-Number>/SPEC-FR-M<Milestone-Number>.<Sub-Number>.md`
2.  **Initial Status**: Set `Status` in the metadata header to `DRAFT`.
3.  **Specs Index Registration**: Open the master [Specs Index](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/INDEX.md), find the corresponding milestone section, and append a new entry for the specification mapping its ID, Title, status `DRAFT`, component, and clickable file link.

---

## Step 3: Present for Review & Approval
Present the drafted specification file to the User for review:
1.  Provide a clickable link to the new spec file (e.g. `[SPEC-FR-M3.X](file:///absolute/path/to/spec.md)`).
2.  Highlight the key technical decisions made in the draft.
3.  **STOP and wait** for the User's explicit feedback or approval. 
4.  **Accepted Transition**: Once approved, move the specification status to `ACCEPTED` inside its own metadata header AND update its status to `ACCEPTED` in the [Specs Index](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/INDEX.md) table before beginning any task execution. (Adheres to Principle P6 of the [Project Constitution](file:///Users/R.Pasquini/Projects/side/tacito-square/specs/constitution.md)).

---

## SPECIFICATION FILE TEMPLATE

```markdown
# SPEC-FR-M{Milestone}.{Number}: {Title}

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M{Milestone}.{Number}               |
| Status        | DRAFT                                       |
| Milestone     | M{Milestone}                                |
| Component     | {keeper|agent|operator|bff|shared|deploy}  |
| Depends On    | SPEC-FR-M{X}.{Y}, ... (or none)             |
| Supersedes    | SPEC-FR-M{X}.{Y}, ... (or none)             |

## Context

{Detailed context describing the problem statement, why this capability is needed, and any business/domain background.}

## Specification

1. {Requirement 1 using RFC 2119: The system MUST...}
2. {Requirement 2: The keeper MUST...}
3. {Requirement 3: The adapter MAY...}

## Acceptance Criteria

1. {Verifiable Criterion 1 (e.g., Domain Model validations, constraints, indexes)}
2. {Verifiable Criterion 2 (e.g., API behavior, HTTP status codes, error payloads)}
3. {Verifiable Criterion 3 (e.g., Decoupling boundaries, hexagonal checks)}

## Test Plan

### Automated Tests
1. **Unit Tests**:
   - {Description of unit test assertions, mocks, and validation scenarios.}
2. **Integration Tests**:
   - {Description of integration/database/HTTP mock assertions.}

### Manual Verification
1. {Steps for manual verification, environment configurations, or deployment validations.}

## API Contract (if applicable)

### Request Format
- Endpoint: `POST /api/v1/...`
- Headers: `Content-Type: application/json`, `X-Tenant-ID: ...`
- Payload:
  ```json
  {
    "name": "string (required)"
  }
  ```

### Response Format
- Status: `201 Created`
- Payload:
  ```json
  {
    "id": "uuid",
    "name": "string"
  }
  ```

## Files Affected

- `[NEW] internal/{component}/domain/...`
- `[MODIFY] internal/{component}/ports/...`
```
