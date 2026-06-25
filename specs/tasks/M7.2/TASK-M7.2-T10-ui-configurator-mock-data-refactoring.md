# TASK-M7.2-T10: UI Configurator Gaps and Mock Data Refactoring

| Field       | Value                                                              |
|-------------|--------------------------------------------------------------------|
| Task ID     | TASK-M7.2-T10                                                      |
| Spec        | SPEC-FR-M7.2                                                       |
| Boundary    | UI Configurator (`ui/configurator/src/`)                           |
| Status      | DRAFT                                                              |
| Depends On  | TASK-M7.2-T9-C                                                     |

## Objective

Align the Configurator React UI with the downstream BFF backend endpoints and schema definitions. Ensure the wizard options, advanced settings, and community assignment flows bind correctly to the live BFF services instead of relying on mock data or fallback variables.

## Detailed Requirements

### 1. Agent Creation Wizard Steps & Fields
* **Brain Selection**: Add form controls to select an existing Brain (LLM Binding) or create a new Brain inline within the wizard.
* **Prompt Selection**: Add a selection list for existing Prompt templates, include file upload support, and support writing/editing a main prompt inline.
* **Long-Term Memory (LTM)**: Introduce a toggle checkbox "Enable Long-Term Memory". Show vector store details (LTM Collection name, LTM Dimension) only when checked.
* **Skills checklist**: Render a list of available Skills for multi-selection, and allow creating/injecting a new Skill inline.
* **MCP Clients toggle**: Display a checkbox for enabling/disabling MCP clients, which must remain read-only/disabled for now.

### 2. Advanced Agent Authoring
* Expand the raw JSON editor / synchronization capability in `RawJsonEditor.tsx` to support editing and authoring referenced resources:
  * Brains (LLM Bindings)
  * Skills
  * MCP Clients / Servers

### 3. Community Authoring & Assignments
* Ensure the Community Form handles topology configurations.
* Refactor the Agent-Community assignment flow to use transactional assignment endpoints (`POST /api/v1/configurator/communities/:id/agents/:agent_id` and `DELETE ...`) rather than passing a mutable static list of agents within the community schema document.

### 4. Live Backend Connection & Mock Data Eviction
* **Critical Issue**: Verify why the deployed UI configurator behaves as if it is using mock data instead of calling live BFF endpoints.
* Ensure API endpoints correctly resolve the host/BFF routes dynamically, session cookies are properly attached, and Keycloak/OIDC flow authentication functions correctly in the deployed environments.
