# SPEC-FR-M4.6: Agent CRD Submission

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M4.6                                |
| Status        | IN_PROGRESS                                 |
| Milestone     | M4                                          |
| Component     | keeper                                      |
| Depends On    | SPEC-FR-M3.7, SPEC-FR-M4.1                 |
| Supersedes    | none                                        |

## Context

When an agent template is declartively assigned to a community, the Keeper must construct and submit a `TacitoAgent` Custom Resource to the Kubernetes API Server. The Operator will then reconcile this resource into running agent pods.

## Specification

1. **Shared Kubernetes API Schema Types (Option A)**:
   * To prevent tight coupling between Keeper and Operator components, the `TacitoAgent` and `TacitoCommunity` Go structs MUST reside in a shared package:
     `pkg/kubernetes/apis/tacito/v1alpha1/`
   * Both `keeper` and `operator` Go code bases MUST import this shared package for custom resource marshaling and client-go communication.

2. **Custom Resource Construction & Multi-Tenancy**:
   * The constructed custom resource MUST use `apiVersion: tacito.square.io/v1alpha1` and `kind: TacitoAgent`.
   * **Multi-Tenancy Mapping**: The client MUST extract the resolved tenant ID from context (`tenant.FromContext(ctx)`) and populate the CR's `spec.tenantId` field to preserve strict isolation boundaries.
   * **Base Configuration Mapping**:
     * `spec.agentName` <- `agent.Name`
     * `spec.communityRef` <- `agent.CommunityID.String()`
     * `spec.llmConfig.model` <- `agent.Brain.Model`
     * `spec.llmConfig.temperature` <- `agent.Brain.Temperature`
     * `spec.llmConfig.maxTokens` <- `agent.Brain.MaxTokens`

3. **System Prompt Resolution & Synthesis**:
   * The CRD coordinator MUST fetch the attached `PromptTemplate` and `Skills` using `PromptRepository` and `SkillRepository` injected into its constructor.
   * The system prompt MUST be dynamically compiled into `spec.systemPrompt` using this structure:
     ```text
     Description: <agent.Description>

     Directives:
     <promptTemplate.Content>

     Skills:
     - <skill_1.Name>: <skill_1.Description>
     - <skill_2.Name>: <skill_2.Description>
     ```

4. **Resilient Outbound Call Policies & Conflict Resolution**:
   * Calls to the Kubernetes API server MUST propagate timeouts via `context.Context` (default: 5 seconds timeout limit).
   * **Conflict Handling**: Wrap the creation/update call in `k8s.io/client-go/util/retry.RetryOnConflict` to handle concurrent Kube-API write conflicts cleanly.
   * **Network Retries**: For transient connection errors, apply an exponential backoff retry loop:
     * `MaxRetries`: 3
     * `BaseBackoff`: 100ms
     * `MaxBackoff`: 2.0 seconds
     * `Jitter`: 20% random backoff scaling.

5. **Asynchronous Progress Events via NATS**:
   * During the background goroutine execution, the CRD coordinator MUST publish progress/error transition events to the NATS bus to let the system reactively observe execution state:
     * **Started**: `agent.provisioning.started` on goroutine startup.
     * **Success**: `agent.provisioning.completed` on successful API server persistence.
     * **Failed**: `agent.provisioning.failed` if all retries/conflicts fail.
   * Payloads MUST be structured JSON:
     ```json
     {
       "tenant_id": "<tenantId>",
       "agent_id": "<agentId>",
       "community_id": "<communityId>",
       "timestamp": "<RFC3339Time>",
       "error": "<errorMessage_if_failed>"
     }
     ```

## Acceptance Criteria

1. **CR Struct Correctness**:
   * The submitted custom resource MUST contain all spec fields correctly filled out, including the active `spec.tenantId` matching the context and the synthesized `spec.systemPrompt`.
2. **Conflict Resolution**:
   * Concurrent requests to assign the same agent MUST resolve cleanly without throwing unhandled `409 Conflict` errors to the caller, thanks to `RetryOnConflict`.
3. **NATS Progress Signaling**:
   * Initiating an assignment MUST trigger a NATS message published to `agent.provisioning.started`.
   * A successful API write MUST trigger `agent.provisioning.completed`.
   * A simulated Kube-API failure (e.g. timeout) MUST result in `agent.provisioning.failed` containing the concrete error message.
4. **Hexagonal Boundaries**:
   * The `AgentService` in the application layer must remain completely unaware of `client-go` or NATS connection primitives, communicating solely through `ports/outbound/CRDCoordinator`.

## Test Plan

### Automated Tests
* **Mock CRD Coordinator & NATS Verification**:
  * Implement mock tests under `internal/keeper/adapters/outbound/crd/crd_coordinator_test.go` using mock client interfaces (`fake.NewSimpleClientset`).
  * Verify `systemPrompt` synthesis logic resolves and maps templates and skills correctly.
  * Verify that NATS events are published to the expected subjects with valid JSON formats under normal and failed execution scenarios.
* **Makefile Integration**:
  * Execute via `make test`.

### Manual Verification
* Assign an agent to a community via HTTP POST `/api/v1/communities/:id/agents/:id`, listen to NATS topic `agent.provisioning.*` via `nats sub "agent.provisioning.*"`, and inspect that events propagate successfully.

## Files Affected

* `internal/keeper/adapters/outbound/crd/crd_coordinator.go` [NEW]
* `internal/keeper/adapters/outbound/crd/crd_coordinator_test.go` [NEW]
* `internal/keeper/bootstrap.go` (Instantiate and inject K8sCRDCoordinator into AgentService)
* `pkg/kubernetes/apis/tacito/v1alpha1/groupversion_info.go` [NEW - Shared with Operator]
* `pkg/kubernetes/apis/tacito/v1alpha1/tacitoagent_types.go` [NEW - Shared with Operator]
* `pkg/kubernetes/apis/tacito/v1alpha1/zz_generated.deepcopy.go` [NEW - Shared with Operator]

