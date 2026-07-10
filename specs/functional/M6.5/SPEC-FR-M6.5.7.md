# SPEC-FR-M6.5.7: Role-Driven System Prompt Templates (Helm ConfigMaps)

| Field       | Value                                                    |
|-------------|----------------------------------------------------------|
| ID          | SPEC-FR-M6.5.7                                           |
| Status      | DRAFT                                                    |
| Milestone   | M6.5                                                     |
| Component   | agent, operator, deploy                                  |
| Depends On  | SPEC-FR-M6.5.1                                           |
| Supersedes  | SPEC-FR-M9.17 (DRAFT)                                   |

## Context

Each agent role (standalone, hub, spoke) has a different cognitive posture:
- **Hub** agents focus on reasoning about which spoke to delegate tasks to, synthesizing results,
  and orchestrating multi-agent workflows. They must NOT attempt to execute tool actions directly.
- **Standalone** agents operate independently, using MCP tools and skills to complete tasks.
- **Spoke** agents are specialised executors within a community, receiving delegated tasks from a
  hub and returning structured results.

This posture is encoded in a **system prompt template** delivered via Kubernetes ConfigMaps
authored in the Helm chart at **deploy time** (not at runtime). Templates may contain Go
`text/template` placeholders for values injected by the Operator reconciler before the agent pod
starts. This approach keeps the role templates out of agent source code and makes them
configurable without re-building the container image.

## Specification

### 1. Helm-Authored ConfigMaps

The Helm chart creates three ConfigMaps in the agent namespace during installation:

| ConfigMap Name                         | Agent Role  | Cognitive Posture Summary |
|----------------------------------------|-------------|---------------------------|
| `tacito-agent-template-standalone`     | standalone  | Direct task execution; use tools and skills to fulfil requests; respond directly to the user. |
| `tacito-agent-template-hub`            | hub         | Read spoke cards; delegate tasks; synthesise responses. Do NOT execute MCP actions. |
| `tacito-agent-template-spoke`          | spoke       | Specialised execution; accept delegated tasks; return structured results. |

Each ConfigMap has a single data key `template.txt` containing the Go `text/template` source.

**Hub template constraints (MUST be enforced in the default content):**
- The template MUST contain explicit instruction not to call MCP action tools directly.
- The template MUST contain instruction to use `list_available_agents` to discover spoke
  capabilities before delegation.
- The template MUST contain instruction to use `delegate_to_agent` to assign tasks.
- The template MUST NOT contain any instruction to use MCP tool names.

**Standalone / Spoke template constraints:**
- MUST contain instructions for direct tool use and response generation.
- MUST NOT instruct the agent to delegate or enumerate spoke agents.

### 2. Template Placeholder Syntax

Templates use standard Go `text/template` syntax. The following placeholders are rendered by the
Operator reconciler at reconciliation time before the rendered result is stored:

| Placeholder        | Value Source                              | Description                  |
|--------------------|-------------------------------------------|------------------------------|
| `{{.AgentID}}`     | `TacitoAgent.metadata.uid`               | Agent UUID                   |
| `{{.AgentName}}`   | `TacitoAgent.metadata.name`              | Agent Kubernetes name        |
| `{{.CommunityID}}` | `community_assignments.community_id`     | Community UUID               |
| `{{.Role}}`        | `community_assignments.role`             | standalone / hub / spoke     |
| `{{.Description}}` | `Agent.Description` (from Keeper)        | Agent description text       |

The Operator renders the template using `text/template.Execute(...)` with a struct containing
these fields. The rendered output is stored in a per-agent ConfigMap:

- **ConfigMap name:** `{agent-name}-system-prompt`
- **Key:** `system-prompt-template.txt`
- **Mount path:** `/etc/tacito/system-prompt-template.txt` (read-only, inside the agent pod)

This rendered file is the **Layer 1** structural template consumed by the cognitive engine's
`compileDynamicSystemPrompt()` function (see SPEC-FR-M6.5.8).

### 3. Custom Override via CRD Field

The `TacitoAgent` CRD spec gains an optional field:

```go
// SystemPromptConfigMapRef, when set, overrides the role-default system prompt template.
// The named ConfigMap must exist in the same namespace and contain a key "template.txt".
// +optional
SystemPromptConfigMapRef string `json:"systemPromptConfigMapRef,omitempty"`
```

Resolution priority:
1. If `spec.systemPromptConfigMapRef` is non-empty, the Operator reads the template from the
   named ConfigMap (key `template.txt`) in the agent's namespace.
2. Otherwise, the Operator uses the role-default ConfigMap (`tacito-agent-template-{role}`).

This allows cluster administrators to provide custom templates for specific agents without
modifying the Helm chart or container image.

### 4. Operator Rendering Steps

During reconciliation the Operator:
1. Determines the agent's effective role from `community_assignments.role`.
2. Determines the template source ConfigMap (custom override or role-default).
3. Fetches the raw template text from the ConfigMap's `template.txt` key.
4. Builds a `TemplateData` struct with values from the CRD and Keeper API response.
5. Renders the template using `text/template.Execute(...)`.
6. Creates or updates the per-agent ConfigMap `{agent-name}-system-prompt` with the rendered
   content under key `system-prompt-template.txt`.
7. Ensures the Deployment's volume mounts reference the per-agent ConfigMap.

### 5. Template Content Guidelines (Normative)

The following guidelines govern the default template content authored in the Helm chart. They are
part of this specification, not implementation commentary.

**All templates:**
- MUST begin with a brief identity statement using `{{.AgentName}}` and `{{.Role}}`.
- MUST include a section placeholder comment indicating where `PropagatedAgentConfig.Directives`
  and activated skill content will be appended at runtime (these are appended by the cognitive
  engine, not by the Operator).
- MUST be written in clear, instruction-following English appropriate for LLM consumption.

**Hub template additional requirements:**
- MUST explicitly state: "You are a routing and orchestration agent. You do not execute actions
  yourself."
- MUST describe the delegation workflow: discover spokes → select spoke → delegate → synthesise.
- MUST reference the `list_available_agents` and `delegate_to_agent` tools by name.

**Standalone / Spoke additional requirements:**
- MUST explicitly state that the agent is expected to use available tools to complete tasks.
- MUST NOT reference delegation tools (`delegate_to_agent`, `list_available_agents`).

## Acceptance Criteria

1. `helm template` output for the Tacito Square chart includes all three role-default ConfigMaps
   (`tacito-agent-template-standalone`, `tacito-agent-template-hub`,
   `tacito-agent-template-spoke`), each containing a non-empty `template.txt` key.
2. The Operator reconciler, when processing a hub-role agent, fetches
   `tacito-agent-template-hub` (unless overridden), renders placeholders with correct values,
   and stores the result in `{agent-name}-system-prompt`.
3. Setting `spec.systemPromptConfigMapRef: my-custom-template` on a `TacitoAgent` causes the
   Operator to read from `my-custom-template` instead of the role-default.
4. All placeholders (`{{.AgentID}}`, `{{.AgentName}}`, `{{.CommunityID}}`, `{{.Role}}`,
   `{{.Description}}`) are rendered with correct, non-empty values in the resulting
   `/etc/tacito/system-prompt-template.txt` file.
5. The default hub template does not contain any instruction to invoke MCP tool names directly.
6. The agent container's volume mount maps `{agent-name}-system-prompt` to
   `/etc/tacito/system-prompt-template.txt`.

## Test Plan

### Unit — Operator template rendering
- Given a fixture `TemplateData` struct and the default hub template source, assert the rendered
  output contains the correct `AgentID`, `AgentName`, and `Role` values.
- Given an invalid Go template string in a ConfigMap, assert the Operator logs an error and
  requeues (does not panic).

### Integration — Operator reconciler
- Trigger a reconciliation for a `spoke`-role TacitoAgent and assert:
  - The `{agent-name}-system-prompt` ConfigMap is created.
  - The content matches the rendered spoke template with correct placeholder values.
- Trigger a reconciliation with `spec.systemPromptConfigMapRef` set to a pre-existing custom
  ConfigMap and assert that the custom template is used.

### Helm
- Run `helm template` and assert the three role-default ConfigMaps are present.
- Run `helm lint` and assert no warnings.

## Files Affected

| File | Change |
|------|--------|
| `deploy/helm/tacito-square/templates/agent-templates-configmap.yaml` | **NEW** — three role-default ConfigMaps |
| `deploy/helm/tacito-square/values.yaml` | **MODIFY** — add template content values for overrideability |
| `pkg/kubernetes/apis/tacito/v1alpha1/tacitoagent_types.go` | **MODIFY** — add `SystemPromptConfigMapRef` optional field |
| `internal/operator/application/service/reconcile_service.go` | **MODIFY** — template resolution, rendering, and per-agent ConfigMap creation |
| `internal/operator/application/service/reconcile_service_test.go` | **MODIFY** — rendering unit tests |
