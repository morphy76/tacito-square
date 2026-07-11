# TASK-M6.5.3.6: K8s CRD Coordinator Integration — Assembled Configuration Resolution

| Field       | Value |
|-------------|-------|
| ID          | TASK-M6.5.3.6 |
| Status      | DRAFT |
| Spec        | SPEC-FR-M6.5.3 |
| Depends On  | TASK-M6.5.3.5 |

## Description

Update the CRD coordinator's configuration assembly mechanism to fetch resolved prompts and set the `Directives` field in `PropagatedAgentConfig` accordingly.

## Work Items

1. **RED Phase**:
   - Update tests in `internal/keeper/adapters/outbound/crd/crd_coordinator_test.go` to mock prompt repository calls for the new multiple prompt/collection fields.
   - Assert that `ResolveAndSynthesizeSystemPrompt` correctly constructs `Directives` using the resolved prompts concatenated by `\n\n`.

2. **GREEN Phase**:
   - Update `internal/keeper/adapters/outbound/crd/crd_coordinator.go`:
     - Update `ResolveAndSynthesizeSystemPrompt` to fetch resolved prompt models via the prompt resolution logic.
     - Concatenate active prompt contents, separated by `\n\n`.
     - Assign this concatenated system-prompt text to the `Directives` field of `PropagatedAgentConfig` as aligned under the consolidation with `SPEC-FR-M6.5.6`.
     - Set the `Description` field to `Agent.Description`.

3. **REFACTOR Phase**:
   - Run tests: `go test ./internal/keeper/adapters/outbound/crd/...`.

## Acceptance Criteria

1. CRD coordinator tests pass GREEN.
2. `ResolveAndSynthesizeSystemPrompt` uses the union-without-duplication prompt resolution service.
3. Concatenated prompts are mapped directly to `PropagatedAgentConfig.Directives`.
