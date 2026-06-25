# TASK-M7.2-T9-C: UI Configurator Resource & Wizard API Integration

| Field       | Value                                                              |
|-------------|--------------------------------------------------------------------|
| Task ID     | TASK-M7.2-T9-C                                                     |
| Spec        | SPEC-FR-M7.2                                                       |
| Boundary    | UI Configurator (`ui/configurator/src/`)                           |
| Status      | VERIFIED                                                           |
| Depends On  | TASK-M7.2-T9-B                                                     |

## Objective

Update the Configurator React UI's API clients, Wizard forms, and Advanced JSON settings screens to integrate with the new BFF endpoints, matching the underlying model structures.

## Files

| File | Action |
|------|--------|
| `ui/configurator/src/` | MODIFY |

## RED Phase

1. **Write React UI Resource Tests**:
   - Write unit/component tests in `ui/configurator/` asserting that forms for Agent and Community creation validate payloads correctly against the actual schema types (e.g. `CreateAgentRequest`, `CreateCommunityRequest`) defined in the Keeper OpenAPI spec.

## GREEN Phase

1. **Adapt React UI**:
   - Update the API client in `ui/configurator/src/` to connect to the composite `wizard/options` endpoint.
   - Update all configuration types, validations, and step-by-step Wizard/Advanced settings screens to use the exact properties defined in the Keeper models.
   - Confirm the React app compiles and passes all UI component tests.

## REFACTOR Phase

- Clean up frontend validation logic, optimize rendering performance for large models, and refine error handling messages when BFF returns validation errors.
