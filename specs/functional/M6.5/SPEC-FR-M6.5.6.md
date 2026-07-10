# SPEC-FR-M6.5.6: PropagatedAgentConfig — Shared Schema & Reconciler Delivery Protocol

| Field       | Value                                                                        |
|-------------|------------------------------------------------------------------------------|
| ID          | SPEC-FR-M6.5.6                                                               |
| Status      | DRAFT                                                                        |
| Milestone   | M6.5                                                                         |
| Component   | shared, keeper, operator, agent                                              |
| Depends On  | SPEC-FR-M6.5.2, SPEC-FR-M6.5.3, SPEC-FR-M6.5.4, SPEC-FR-M6.5.5            |
| Supersedes  | none                                                                         |

## Context

The cognitive engine currently parses a JSON struct `PropagatedAgentConfig` from the system
prompt, but this struct is undocumented and only exists in the agent's application service
(`internal/agent/application/service/cognitive_engine_types.go`). There is no shared,
versioned definition that Keeper or the Operator can reference when assembling or delivering
this payload.

This spec formalizes the schema as a versioned shared Go contract in `pkg/shared/agent_config/`,
defines how Keeper assembles the struct from resolved prompt and skill collections, and prescribes
how the Operator reconciler delivers it to the agent pod as a Kubernetes ConfigMap mounted as a
file — replacing the current practice of embedding it inside the system-prompt environment variable.

## Specification

### 1. Shared Schema (`pkg/shared/agent_config/`)

Define `PropagatedAgentConfig` and `SkillConfig` as versioned shared Go structs importable by
both the Keeper and the agent without introducing circular dependencies:

```go
// Package agentconfig defines the versioned shared contract for the agent runtime configuration
// assembled by Keeper and delivered by the Operator.
package agentconfig

// CurrentVersion is the schema version. Bump when the struct shape changes in a
// backward-incompatible way.
const CurrentVersion = "1.0"

// PropagatedAgentConfig is the complete configuration payload assembled by Keeper and
// delivered to the agent pod as /etc/tacito/agent-config.json.
type PropagatedAgentConfig struct {
    Version     string        `json:"version"`     // schema version, e.g. "1.0"
    Description string        `json:"description"` // agent description from Agent.Description
    Directives  string        `json:"directives"`  // concatenated prompt contents (resolved order)
    Skills      []SkillConfig `json:"skills"`      // resolved, deduplicated skill list
}

// SkillConfig is a single skill entry within PropagatedAgentConfig.
type SkillConfig struct {
    Name        string `json:"name"`
    Description string `json:"description"`
    Content     string `json:"content"` // full procedural content injected into the system prompt
}
```

Rules:
- `Version` MUST be set to `agentconfig.CurrentVersion` by the assembler.
- The schema version MUST be incremented when the structure changes in a backward-incompatible
  manner. Minor additive changes increment the minor part (`1.1`, `1.2`, ...); breaking changes
  increment the major part (`2.0`).
- Both Keeper (`internal/keeper/...`) and agent (`internal/agent/...`) import ONLY from
  `pkg/shared/agent_config/`. Neither package duplicates the struct.

### 2. Assembly (Keeper — `AgentConfigAssembler`)

A new application service `AgentConfigAssembler` in
`internal/keeper/application/service/agent_config_assembler.go` is responsible for constructing
the `PropagatedAgentConfig` for a given agent at reconciliation time. It depends on outbound port
interfaces only — never on concrete adapters.

Assembly steps:

**a. Resolve effective prompt list** (per SPEC-FR-M6.5.3 union-without-duplication semantics):
  1. Fetch the agent's own `PromptCollection` (if any).
  2. Fetch the community's `PromptCollection` (if any).
  3. Merge: agent-level prompts take precedence; community-level prompts that share a name with an
     agent-level prompt are dropped.
  4. Concatenate active prompt `Content` fields in declared order, separated by `\n\n`.
  5. Assign the concatenated result to `Directives`.

**b. Resolve effective skill list** (per SPEC-FR-M6.5.4 union-without-duplication semantics):
  1. Fetch the agent's own `SkillCollection` (if any).
  2. Fetch the community's `SkillCollection` (if any).
  3. Merge: agent-level skills take precedence; community-level skills sharing a `Name` with an
     agent-level skill are dropped.
  4. Map each resolved `Skill` domain model to `SkillConfig{Name, Description, Content}`.
  5. Assign the resulting slice to `Skills`.

**c. Set `Description`** from `Agent.Description`.

**d. Serialize** the assembled struct to compact JSON using `encoding/json`.

**e. Role-based filtering:**
  - If the agent's effective role (from `community_assignments.role`) is `hub`:
    - **Include** skills in `Skills` (routing procedural knowledge is required).
    - **Exclude** MCP client info — MCP client configurations are NOT included in
      `PropagatedAgentConfig`. They are handled separately in `mcp-clients.json` (see
      SPEC-FR-M6.5.9) and are always an empty array for hub agents.
  - If the role is `standalone` or `spoke`:
    - Include both skills; MCP info lives in a separate file (see SPEC-FR-M6.5.9).

### 3. Delivery (Operator to Agent Pod)

The Operator reconciler delivers the assembled `PropagatedAgentConfig` JSON to the agent pod via
a **Kubernetes ConfigMap mounted as a file**:

- **ConfigMap name:** `{agent-name}-config`
- **Namespace:** same namespace as the TacitoAgent CR.
- **Key inside ConfigMap:** `agent-config.json`
- **Mount path inside the container:** `/etc/tacito/agent-config.json` (read-only volume mount).

The Operator creates or updates this ConfigMap during every reconciliation cycle **before**
applying changes to the Deployment (see SPEC-FR-M6.5.9 for full reconciliation order).

On update the Operator uses a server-side apply or `CreateOrUpdate` pattern to avoid races with
the Kubernetes ConfigMap controller.

### 4. Agent Startup — Reading the Config

On startup the agent reads `/etc/tacito/agent-config.json` and unmarshals it into
`agentconfig.PropagatedAgentConfig`.

- If the file is absent or unparseable, the agent MUST log a fatal error and exit (the pod will
  be restarted by Kubernetes, which will retry reconciliation).
- The agent MUST validate that `Version` matches a version it supports. If the version is
  unsupported, the agent logs an error and exits.
- SIGHUP handling for live-reload is **not** required in this milestone but the file path and
  mechanism MUST be designed to support it in a future spec without breaking changes.

### 5. Role-Based Filtering Summary

| Content               | hub | spoke | standalone |
|-----------------------|-----|-------|------------|
| Description           | yes | yes   | yes        |
| Directives (prompts)  | yes | yes   | yes        |
| Skills                | yes | yes   | yes        |
| MCP client info       | no  | yes*  | yes*       |

*MCP client info is delivered in `/etc/tacito/mcp-clients.json`, not inside
`PropagatedAgentConfig`.

### 6. Brain Configuration Delivery

Brain parameters (LLM provider, model, temperature, max tokens, timeout, API key) are delivered
as individual environment variables — not inside `PropagatedAgentConfig`. See SPEC-FR-M6.5.9.

## Acceptance Criteria

1. The `PropagatedAgentConfig` and `SkillConfig` Go structs are defined in
   `pkg/shared/agent_config/propagated_config.go` and can be imported by both `internal/keeper`
   and `internal/agent` without circular import errors.
2. Keeper's `AgentConfigAssembler` correctly applies union-without-duplication merging for both
   prompts and skills, with agent-level items taking precedence over community-level items.
3. The assembled `PropagatedAgentConfig.Version` field equals `agentconfig.CurrentVersion`.
4. The Operator reconciler creates/updates the `{agent-name}-config` ConfigMap with the correct
   JSON content on each reconciliation.
5. The agent reads and parses the ConfigMap file at startup; if the file is absent the agent
   exits with a fatal error.
6. A hub agent's `PropagatedAgentConfig.Skills` is populated; MCP client data is not present in
   the config file (it is an empty array in `mcp-clients.json`).
7. A standalone/spoke agent's `PropagatedAgentConfig.Skills` is populated; MCP client data is in
   the separate `mcp-clients.json` file.

## Test Plan

### Unit — Keeper assembly
- Given a mock agent with 2 prompts and a community with 3 prompts (1 shared name), assert
  `Directives` contains exactly 4 distinct prompts concatenated in order.
- Given a mock agent with 1 skill and a community with 2 skills (1 shared name), assert
  `Skills` has 2 entries with agent-level taking precedence.
- Assert `Version == agentconfig.CurrentVersion` in all assembled outputs.

### Unit — Agent startup
- Given a valid `agent-config.json` file, assert the agent parses it and the cognitive engine
  receives a correctly populated `PropagatedAgentConfig`.
- Given a missing file, assert the startup function returns a fatal error.
- Given a JSON file with an unsupported `version` value, assert the startup function returns an
  error.

### Integration — Operator reconciler
- Using envtest or a real cluster, trigger a reconciliation of a `TacitoAgent` CRD and assert
  the `{agent-name}-config` ConfigMap exists with the expected JSON content.
- Assert that a second reconciliation **updates** the ConfigMap (does not create a duplicate).

## Files Affected

| File | Change |
|------|--------|
| `pkg/shared/agent_config/propagated_config.go` | **NEW** — shared schema (`PropagatedAgentConfig`, `SkillConfig`, `CurrentVersion`) |
| `pkg/shared/agent_config/propagated_config_test.go` | **NEW** — JSON round-trip tests |
| `internal/keeper/application/service/agent_config_assembler.go` | **NEW** — assembly logic |
| `internal/keeper/application/service/agent_config_assembler_test.go` | **NEW** — unit tests |
| `internal/keeper/application/ports/outbound/agent_config_assembler_port.go` | **NEW** — outbound port interface |
| `internal/operator/application/service/reconcile_service.go` | **MODIFY** — create/update ConfigMap during reconciliation |
| `internal/agent/application/service/cognitive_engine.go` | **MODIFY** — read from `/etc/tacito/agent-config.json` instead of env var |
| `internal/agent/application/service/cognitive_engine_types.go` | **MODIFY** — replace local struct with import from `pkg/shared/agent_config` |
