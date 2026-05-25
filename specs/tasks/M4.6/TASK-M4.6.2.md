# TASK-M4.6.2: System Prompt Synthesis & Multi-Tenancy Mapping

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M4.6.2                                 |
| Status        | PLANNED                                     |
| Spec          | SPEC-FR-M4.6                                |
| Depends On    | TASK-M4.6.1                                 |

## Description

Implement the core logic inside the Keeper CRD coordinator that fetches the agent's prompt template and skills out-of-band and synthesizes them into a compiled string for `spec.systemPrompt`. The coordinator must fetch these details using Prompt and Skill repositories injected into its constructor and map the active multi-tenancy `tenantId` context accurately.

## Boundary & Target Functions

- **Package**: `internal/keeper/adapters/outbound/crd`
- **File**: `internal/keeper/adapters/outbound/crd/crd_coordinator.go`
- **Target Functions**:
  - `(c *K8sCRDCoordinator) ResolveAndSynthesizeSystemPrompt(ctx context.Context, agent *model.Agent) (string, error)`

## Work Items

1. **RED Phase**:
   * Implement unit tests in `internal/keeper/adapters/outbound/crd/crd_coordinator_test.go` to assert:
     * `ResolveAndSynthesizeSystemPrompt` accurately queries mock repos.
     * The resulting prompt maps the agent description, custom directives template, and skills lists (names and descriptions) matching:
       ```text
       Description: <desc>

       Directives:
       <directiveContent>

       Skills:
       - <skillName>: <skillDesc>
       ```
     * Submission payload has the active context `tenantId` set inside `spec.tenantId`.

2. **GREEN Phase**:
   * Inject `PromptRepository` and `SkillRepository` into `K8sCRDCoordinator` constructor.
   * Implement `ResolveAndSynthesizeSystemPrompt` to fetch PromptTemplate and Skill arrays sequentially.
   * Perform string compilation and mapping to custom resources.

3. **REFACTOR Phase**:
   * Optimize database lookups to prevent excessive queries (e.g. cache lookups or map resolution loops).

## Acceptance Criteria

1. Prompt synthesis unit tests pass successfully.
2. Context tenant ID is cleanly mapped without mutations.
