---
trigger: manual
description: Interactively drafting new GitHub functional specification issues with the user.
---

# SDD Workflow: Spec Creation (Interactive Drafting)

This workflow defines the step-by-step interactive process for creating a new **Functional Specification (FR)** as a GitHub Issue, fully aligned with the [Project Constitution](specs/constitution.md) and `.github/ISSUE_TEMPLATE/spec.yml`.

---

## Step 1: The Interactive Interview Phase
Governed by Principle P1 (**Spec-Driven Development**) of the [Project Constitution](specs/constitution.md), the agent MUST NOT draft or write a specification in isolation. You must conduct a structured, interactive interview with the User to gather precise details. Ask questions focusing on:

1.  **Scope & Context**: What is the core business problem being solved? What is the background context?
2.  **Bounded Context & Component**: Which architectural component is affected (`keeper`, `agent`, `operator`, `bff`, `ui`, `shared`, `deploy`)?
3.  **RFC 2119 Specifications**: What are the strict `MUST`, `SHOULD`, and `MAY` requirements?
4.  **Acceptance Criteria**: What are the concrete, verifiable conditions for success?
5.  **Test Plan & API Contracts**: How will we verify this? What are the REST payloads, status codes, or NATS topics?
6.  **Milestones & Dependencies**: Which milestone does this target? Does it depend on other issues?

---

## Step 2: GitHub Issue Creation
Once the User has provided sufficient details, create the GitHub Issue using the standard template:

```bash
gh issue create \
  --title "[SPEC]: <Title>" \
  --label "type:spec,status:draft,comp:<component>" \
  --body "## Context
<Context description>

## Specification
1. The system MUST ...

## Acceptance Criteria
1. ...

## Test Plan
### Unit Tests
- ...

### Integration Tests
- ...

## API Contract (if applicable)
...

## Files Affected
- ...
"
```

---

## Step 3: Present for Review & Approval
Present the drafted issue to the User:
1.  Provide the GitHub Issue number and link.
2.  Highlight the key technical decisions made in the draft.
3.  **STOP and wait** for the User's explicit review and feedback.
4.  Once approved, transition the issue label to `status:accepted` per `sdd-spec-review.md`.
