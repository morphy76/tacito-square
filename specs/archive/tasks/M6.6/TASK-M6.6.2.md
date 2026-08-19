# TASK-M6.6.2: Implement Hub-Side Handoff Detection and Execution

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M6.6.2                                 |
| Status        | VERIFIED                                    |
| Spec          | SPEC-FR-M6.6                                |
| Depends On    | TASK-M6.6.1                                 |

## Description

Implement handoff detection inside the Hub orchestrator. If a Spoke agent responds with a `suggest_handoff` JSON action:
1. Parse the JSON and extract `target` and `reason`.
2. Validate if the target Spoke exists via `AgentDiscovery.GetCards()`.
3. If valid, log the suggestion in the Hub's STM: `[Observation] Spoke Agent '<spoke_name>' suggested handoff to '<target_name>' because: <reason>`.
4. Construct the concatenated delegation message: `[Handoff instruction: <reason>] Original task: <original_message>`, where `original_message` is retrieved from `state.PendingSpokes[payload.AgentName]`.
5. Populate `ContextHistory` in `AgentDelegationPayload` with recent turns (up to 15) from the Hub's STM.
6. Publish the task to the target Spoke NATS subject and update the `OrchestrationState` in Redis (swap pending spokes and save).
7. If the target is invalid or missing, fall back to the normal reasoning loop (no handoff execution, proceed with normal observation).

## Work Items

1. **RED Phase**:
   - Write a unit test in [orchestrator_test.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/application/service/orchestrator_test.go) that simulates a Spoke returning a valid `suggest_handoff` JSON response.
   - Assert that the Hub parses this correctly, logs it in STM, updates `OrchestrationState` in Redis (removing delegator, adding target), and publishes `AgentDelegationPayload` to NATS with the concatenated message and `ContextHistory`.
   - Write another unit test simulating a missing/invalid handoff target, verifying that the Hub falls back to `runOrchestrationTurn`.
2. **GREEN Phase**:
   - Define a `HandoffSuggestion` struct for JSON parsing.
   - Modify `ProcessSpokeResponse` in [orchestrator.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/application/service/orchestrator.go) to perform handoff suggestion detection, verification against active agent cards, logging to STM, state updating, and publishing.
   - Run tests and fix compiler issues until tests pass.
3. **REFACTOR Phase**:
   - Clean up JSON parsing heuristics (e.g. Markdown code fences trimming).
   - Ensure clear logging via `zerolog` for both successful handoffs and invalid target fallbacks.

## Acceptance Criteria

1. Handoff suggestions are parsed and validated asynchronously.
2. Valid handoff requests publish `AgentDelegationPayload` containing `ContextHistory` and the concatenated task instructions.
3. Hub STM is updated with a human-readable observation of the handoff event.
4. If target validation fails, fallback reasoning executes successfully.
