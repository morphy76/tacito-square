# TASK-M7.2-T4: Wizard-driven CRUD & Advanced Settings Panel

| Field       | Value                                                              |
|-------------|--------------------------------------------------------------------|
| Task ID     | TASK-M7.2-T4                                                      |
| Spec        | SPEC-FR-M7.2                                                       |
| Boundary    | UI Configurator (`ui/configurator/`)                               |
| Status      | VERIFIED                                                           |
| Depends On  | SPEC-FR-M7.2, TASK-M7.2-T1, TASK-M7.2-T2, TASK-M7.2-T3             |

## Objective

Create the administration and editing views for managing agents, communities, and agent-community assignments targeting the BFF route namespace `/api/v1/configurator/`. For basic usability, provide a step-by-step Wizard-driven interface guiding the creation and updates of these components. For advanced administrators, integrate an Advanced Settings panel that exposes low-level raw schema/JSON editing directly.

## Files

| File | Action |
|------|--------|
| `ui/configurator/src/components/Wizard/AgentWizard.tsx` | NEW |
| `ui/configurator/src/components/Wizard/CommunityWizard.tsx` | NEW |
| `ui/configurator/src/components/AdvancedSettings/RawJsonEditor.tsx` | NEW |
| `ui/configurator/src/components/AgentCRUD/` (various list/form files) | NEW |
| `ui/configurator/src/components/CommunityCRUD/` (various list/form files) | NEW |
| `ui/configurator/src/components/AssignmentCRUD/` (various list/form files) | NEW |
| `ui/configurator/src/components/AgentCRUD/AgentForm.test.tsx` | NEW |
| `ui/configurator/src/components/CommunityCRUD/CommunityForm.test.tsx` | NEW |

## RED Phase

1. **Form Validation & Wizard Test Suite**:
   - Create `ui/configurator/src/components/AgentCRUD/AgentForm.test.tsx` and assert that:
     - Form fields (e.g. name, description, model) display validation/required errors when submitted empty.
     - The Agent Wizard properly displays steps (e.g. Step 1: Metadata, Step 2: System Prompt, Step 3: Skills) and validates inputs before letting the user click "Next".
     - Submitting the form calls `POST /api/v1/configurator/agents` with the correct payload and handles success by redirecting or showing a success indicator.
     - API error responses (e.g. 422 validation failure from BFF) are caught and displayed to the user as readable alerts.
   - Create `ui/configurator/src/components/CommunityCRUD/CommunityForm.test.tsx` and assert equivalent validation and submission behavior for communities.
   - Run Vitest tests (`npm run test`) inside `ui/configurator/` (must fail because components do not exist yet).

2. **Advanced Raw JSON Editor Tests**:
   - Create tests in `RawJsonEditor.test.tsx` asserting:
     - Typing invalid JSON strings triggers a validation error message in the UI and disables the "Save" button.
     - Valid JSON string can be successfully parsed, displaying no errors and enabling the "Save" button.
     - Clicking "Save" calls the backend API with the parsed object.

## GREEN Phase

1. **Implement Wizard Components**:
   - Create step-by-step wizard components for Agents (`AgentWizard.tsx`) and Communities (`CommunityWizard.tsx`).
   - Group fields logically (e.g. metadata, configuration properties, assignments).
   - Style with progress indicators, smooth transitions, and premium back/next controls using the Piazza Tacito CSS theme variables.

2. **Implement Advanced Settings Panel**:
   - Create `RawJsonEditor.tsx` with a standard styled textarea or code editor mock.
   - Provide live validation checking of text as the user edits, using standard `JSON.parse` catching syntax errors.
   - Render helpful syntax error alerts if parsing fails.

3. **Implement List and Detail Screens (CRUD)**:
   - Build tables/grids displaying all agents and communities.
   - Provide "Edit", "Delete", and "View details" buttons.
   - Add assignment forms allowing users to link agents to communities by selecting from dropdown arrays.
   - Wire backend API request logic:
     - Retrieve configurations from `GET /api/v1/configurator/agents` / `GET /api/v1/configurator/communities`.
     - Persist via `POST` (create), `PUT` (update), and `DELETE` (remove).

4. **Verify tests**:
   - Run tests (`npm run test`) and ensure all form and editor tests pass.

## REFACTOR Phase

- Extract shared Wizard layouts and stepper bars into a common component.
- Ensure that the JSON editor validates against a schema (if available) or implements debounced change validation.
- Verify that standard navigation (e.g., using `react-router-dom` or similar) handles back button behaviors correctly during wizard completion.
