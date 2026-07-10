# BUG-M7.2: UI Configurator Gaps in Wizard Binding, Advanced Authoring, and Assignment Workflows

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M7.2                                                           |
| Status        | CLOSED                                                             |
| Severity      | HIGH                                                               |
| Milestone     | M7 — BFF & UIs                                                     |
| Affects       | `ui/configurator/src/App.tsx`, `ui/configurator/src/components/Wizard/AgentWizard.tsx`, `ui/configurator/src/components/Wizard/CommunityWizard.tsx`, `ui/configurator/src/components/CommunityCRUD/CommunityForm.tsx` |
| Violates      | SPEC-FR-M7.2                                                       |
| Discovered    | Code inspection of frontend forms and comparison with BFF / Keeper OpenAPI and Go structs |

## Problem Statement

The Tacito Square Configurator UI is decoupled from the BFF, but there are structural gaps where UI elements either ignore, misrepresent, or fail to bind to actual downstream BFF capabilities. Specifically:

1.  **Agent Creation Through Wizard Gaps:**
    The BFF/Keeper backend's `CreateAgentRequest` accepts `MCPClients` (list of MCP client configs) and `Deployment` (deployment request tier constraints). However, the React `AgentWizard.tsx` and `AgentForm.tsx` components do not provide any visual elements or form fields to configure these settings, leaving spawned agents with default empty configurations.
2.  **Advanced Authoring for Tacito Resources:**
    The Advanced settings view (`RawJsonEditor.tsx`) only allows editing the currently active tab's collections (agents or communities lists). It does not support synchronizing or authoring other essential Tacito configuration resources (like LLM Bindings or Prompt Templates) which are read-only. Furthermore, the advanced sync does not support synchronizing the entire workspace schema (both agents and communities together), causing potential sync conflicts when updating relational states.
3.  **Assign/Unassign Workflow Mismatches:**
    *   The BFF provides discrete transactional endpoints for community agent assignments: `POST /api/v1/configurator/communities/:id/agents/:agent_id` and `DELETE /api/v1/configurator/communities/:id/agents/:agent_id`.
    *   In the UI configurator's Community Wizard (`CommunityWizard.tsx` and `CommunityForm.tsx`), a static `agents` array is exposed and editable. When submitting the community form, the payload contains this array. However, the BFF's `CreateCommunityRequest` / `UpdateCommunityRequest` payloads do *not* accept or parse an `agents` list.
    *   Consequently, assigning or unassigning agents inside the Community Wizard form is silently ignored on save, resulting in state synchronization loss unless the user specifically uses the decoupled dropdown assignment controls on the dashboard.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| Agent Wizard / Forms | [AgentWizard.tsx](file:///Users/R.Pasquini/Projects/side/tacito-square/ui/configurator/src/components/Wizard/AgentWizard.tsx), [AgentForm.tsx](file:///Users/R.Pasquini/Projects/side/tacito-square/ui/configurator/src/components/AgentCRUD/AgentForm.tsx) | Missing configuration fields for `mcp_clients` and `deployment.tier`. |
| Community Wizard / Forms | [CommunityWizard.tsx](file:///Users/R.Pasquini/Projects/side/tacito-square/ui/configurator/src/components/Wizard/CommunityWizard.tsx), [CommunityForm.tsx](file:///Users/R.Pasquini/Projects/side/tacito-square/ui/configurator/src/components/CommunityCRUD/CommunityForm.tsx) | Exposes `agents` arrays in forms which are silently discarded by the BFF community endpoints on submit. |
| Dashboard Controller | [App.tsx](file:///Users/R.Pasquini/Projects/side/tacito-square/ui/configurator/src/App.tsx) | Does not unify community and agent configurations in Advanced Sync, and lacks handling for full relational state synchronization. |

## Impact

1.  **Deployment Incomplete:** Users cannot provision agents with specialized hardware tiers (e.g. GPU vs CPU) or configure their connected MCP tools from the graphical interface.
2.  **State Inconsistency:** Saving a community specification via the Wizard resets or discards agent assignments if the user expects the wizard's `agents` field to save, leading to configuration confusion and broken community topologies.

## Expected Behaviour

1.  **Agent Wizard:** The Agent Wizard/Form MUST provide fields to optionally configure deployment tier (e.g., `tier` select dropdown) and associate MCP clients.
2.  **Assignment Integrity:** The Community Wizard/Form MUST NOT present a mutable list of assigned agents as part of the community document payload if it cannot be saved. Instead, community creation/modification should focus purely on community metadata, forcing membership associations to utilize the dedicated, transactional `/communities/:id/agents/:agent_id` endpoints.
3.  **Advanced Sync:** Advanced settings should clarify schema synchronization constraints or allow full-workspace relational schema authoring.

## Acceptance Criteria

1.  A BUG document `BUG-M7.2.md` is created and indexed.
2.  UI components are refactored to align wizard fields and assignments with correct BFF routes.
3.  State integrity is preserved during creation and editing of communities and agents.
