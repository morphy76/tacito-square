# SPEC-FR-M6.5.9: Agent Environment Setup on Community Assignment

| Field       | Value                                                          |
|-------------|----------------------------------------------------------------|
| ID          | SPEC-FR-M6.5.9                                                 |
| Status      | DRAFT                                                          |
| Milestone   | M6.5                                                           |
| Component   | operator, keeper                                               |
| Depends On  | SPEC-FR-M6.5.1, SPEC-FR-M6.5.6, SPEC-FR-M6.5.7               |
| Supersedes  | none                                                           |

## Context

When an agent is assigned to a community and the Operator reconciles the `TacitoAgent` CRD, it
must materialise a complete, self-contained runtime environment for the agent pod. This
environment spans multiple Kubernetes primitives: environment variables (for configuration
scalars), Secrets (for credentials), and ConfigMap volume mounts (for structured config files and
system prompt templates).

Currently the Operator does not have a formally specified, ordered reconciliation contract. This
spec defines the complete environment layout — what goes where, why, and in what order — so that
all Operator implementations and tests agree on a single canonical contract.

## Specification

### 1. Environment Variables (Operator injects into pod spec)

The following environment variables MUST be set on the agent container spec. All values are
sourced by the Operator reconciler from Keeper API responses and the `TacitoAgent` CRD:

| Env Var                   | Source                                               | Type    | Description                              |
|---------------------------|------------------------------------------------------|---------|------------------------------------------|
| `TS_AGENT_ROLE`           | `community_assignments.role`                         | string  | `standalone` / `hub` / `spoke`           |
| `TS_AGENT_COMMUNITY_ID`   | `community_assignments.community_id`                 | string  | UUID of the agent's community            |
| `TS_AGENT_LLM_PROVIDER`   | `LLMBinding.Provider`                                | string  | `openai`, `ollama`, etc.                 |
| `TS_AGENT_LLM_BASE_URL`   | `LLMBinding.APIBaseURL`                              | string  | Provider REST base URL                   |
| `TS_AGENT_LLM_MODEL`      | `LLMBinding.DefaultModel`                            | string  | Model identifier (e.g. `gpt-4o`)        |
| `TS_AGENT_LLM_TEMPERATURE`| `Agent.Brain.Temperature` (fallback: `LLMBinding.DefaultTemperature`) | float64 | Sampling temperature |
| `TS_AGENT_LLM_MAX_TOKENS` | `Agent.Brain.MaxTokens` (fallback: `LLMBinding.DefaultMaxTokens`)     | int     | Max completion tokens |
| `TS_AGENT_LLM_TIMEOUT`    | `LLMBinding.TimeoutSeconds`                          | int     | Seconds before LLM request timeout       |
| `TS_AGENT_STM_NAMESPACE`  | `Agent.ShortTermMemory.KeyNamespace`                 | string  | Redis key prefix for STM isolation       |
| `TS_AGENT_STM_TTL`        | `Agent.ShortTermMemory.TTLSeconds`                   | int     | Redis key TTL in seconds                 |
| `TS_AGENT_TIER`           | `Agent.Tier`                                         | string  | Deployment tier identifier               |

**Fallback rules:**
- If `Agent.Brain.Temperature` is unset (zero value), use `LLMBinding.DefaultTemperature`.
- If `Agent.Brain.MaxTokens` is unset (zero value), use `LLMBinding.DefaultMaxTokens`.

**Agent ID** (`TS_AGENT_ID`) is already injected by the Operator from `TacitoAgent.metadata.uid`
and is not repeated here.

### 2. Kubernetes Secrets (mounted via `secretKeyRef`)

Credentials MUST never be stored as plain-text environment variable literals. They are sourced
from Kubernetes Secrets via `secretKeyRef` entries in the pod spec:

| Env Var              | Secret Name                                   | Secret Key  | Description                     |
|----------------------|-----------------------------------------------|-------------|----------------------------------|
| `TS_AGENT_LLM_API_KEY` | `LLMBinding.APIKeySecretRef` (name field)  | `api-key`   | LLM provider API key             |

For MCP client authentication tokens: each `MCPClient` domain record has an optional
`AuthSecretRef` field. If set, the Operator injects an env var
`TS_MCP_{CLIENT_NAME_UPPER}_AUTH_TOKEN` sourced from the named Secret's `auth-token` key. If
`AuthSecretRef` is unset for a given client, no env var is injected for that client's auth.

### 3. ConfigMap Mounts (files mounted into the agent container)

Three ConfigMaps are mounted as read-only files at well-known paths:

| ConfigMap Name               | Mount Path                                 | Content                                               |
|------------------------------|--------------------------------------------|-------------------------------------------------------|
| `{agent-name}-system-prompt` | `/etc/tacito/system-prompt-template.txt`   | Rendered role template (from SPEC-FR-M6.5.7)          |
| `{agent-name}-config`        | `/etc/tacito/agent-config.json`            | `PropagatedAgentConfig` JSON (from SPEC-FR-M6.5.6)    |
| `{agent-name}-mcp-clients`   | `/etc/tacito/mcp-clients.json`             | JSON array of MCP client configs (empty `[]` for hub) |

All three ConfigMaps use `defaultMode: 0444` (world-readable, read-only).

#### `mcp-clients.json` Schema

```json
[
  {
    "name": "string",
    "transport": "stdio|sse|streamable-http",
    "endpoint": "string",
    "auth_token_env": "TS_MCP_{CLIENT_NAME_UPPER}_AUTH_TOKEN"
  }
]
```

- `auth_token_env` is the name of the environment variable containing the auth token (see §2).
  If the client has no auth, this field is an empty string.
- For hub agents, this file contains an empty JSON array `[]`.

### 4. Reconciliation Order

The Operator reconciler MUST apply changes in the following strict order to prevent partial or
inconsistent agent environments:

1. **Resolve dependencies from Keeper API** (in parallel where possible):
   - LLM binding record for the agent's community.
   - Resolved prompt list (per SPEC-FR-M6.5.3).
   - Resolved skill list (per SPEC-FR-M6.5.4).
   - MCP client list (for non-hub agents).
   - Agent record (for `Brain`, `ShortTermMemory`, `Tier`, `Description`).

2. **Assemble `PropagatedAgentConfig`** via `AgentConfigAssembler` (per SPEC-FR-M6.5.6).

3. **Create or update ConfigMaps** (in any order; all three must succeed before proceeding):
   - `{agent-name}-config` — `PropagatedAgentConfig` JSON.
   - `{agent-name}-mcp-clients` — MCP client array JSON.

4. **Render system prompt template** and create or update `{agent-name}-system-prompt` ConfigMap
   (per SPEC-FR-M6.5.7).

5. **Update the Deployment** with:
   - All env vars from §1.
   - All `secretKeyRef` entries from §2.
   - All volume mounts and volumes referencing the three ConfigMaps from §3.

6. **Update assignment record** via Keeper API: set
   `community_assignments.informed_at = NOW()` on success.

This order guarantees that the pod spec always references ConfigMaps that already exist, avoiding
pod startup failures due to missing volumes.

### 5. Failure Handling

- If any Keeper API call in step 1 fails (network error, 4xx, 5xx), the reconciler MUST log the
  error at `error` level using zerolog and return an error to the controller-runtime reconcile
  loop, which will requeue with exponential backoff.
- If any ConfigMap create/update in steps 3–4 fails, the reconciler MUST return an error and
  requeue. The Deployment MUST NOT be updated until all ConfigMaps succeed.
- If the Deployment update (step 5) fails, the reconciler logs and requeues. The
  `community_assignments.informed_at` field is NOT updated until step 5 succeeds.
- The reconciler MUST be idempotent: running it multiple times with the same inputs produces the
  same Kubernetes resource state.

### 6. RBAC Requirements

The Operator's `ServiceAccount` MUST have the following permissions for this reconciliation to
succeed (to be declared in the Helm chart):

```yaml
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get", "list", "watch", "create", "update", "patch"]
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps"]
  resources: ["deployments"]
  verbs: ["get", "list", "watch", "create", "update", "patch"]
```

## Acceptance Criteria

1. A newly assigned agent pod has all environment variables from §1 set to correct, non-empty
   values.
2. `TS_AGENT_LLM_API_KEY` is sourced from a `secretKeyRef` in the pod spec — never as a
   literal string value.
3. A hub agent's `/etc/tacito/mcp-clients.json` file contains exactly `[]` (empty JSON array).
4. A standalone or spoke agent's `/etc/tacito/mcp-clients.json` contains one entry per attached
   MCP client with correct `name`, `transport`, `endpoint`, and `auth_token_env` fields.
5. The `community_assignments.informed_at` timestamp is updated via Keeper API after successful
   reconciliation (step 6).
6. If the Keeper API call in step 1 fails, the Deployment is NOT updated and the reconciler
   requeues.
7. Running the reconciler twice with the same inputs does not create duplicate ConfigMaps or
   Deployment revisions.

## Test Plan

### Unit — Operator reconciler (with mock Keeper)
- Mock Keeper returns a fixed LLM binding, 2 skills, 3 prompts, and 2 MCP clients.
- Assert that the generated pod spec contains all expected env vars with correct values.
- Assert that the pod spec contains `secretKeyRef` entries for `TS_AGENT_LLM_API_KEY`.
- Assert hub-role: `mcp-clients.json` ConfigMap data key contains `[]`.
- Assert spoke-role: `mcp-clients.json` ConfigMap data key contains 2 MCP client objects.

### Integration — End-to-end reconciler
- Deploy a `TacitoAgent` CRD in a test namespace with a community assignment.
- Assert all three ConfigMaps are created with correct content.
- Assert the Deployment's volume mounts reference the correct ConfigMaps.
- Assert `kubectl exec` into the agent container can read all three mounted files.

### Manual verification
- `kubectl exec` into a running agent pod and run:
  - `printenv | grep TS_AGENT_` — verify all required vars.
  - `cat /etc/tacito/agent-config.json | jq .version` — verify JSON is valid.
  - `cat /etc/tacito/mcp-clients.json` — verify content matches role.

## Files Affected

| File | Change |
|------|--------|
| `internal/operator/application/service/reconcile_service.go` | **MODIFY** — full env/secret/configmap assembly per this spec |
| `internal/operator/application/service/reconcile_service_test.go` | **MODIFY** — unit tests for env assembly and role-based mcp-clients content |
| `internal/operator/adapters/outbound/keeper_client.go` | **NEW or MODIFY** — Keeper API client (fetch LLM binding, prompts, skills, MCP clients) |
| `deploy/helm/tacito-square/templates/operator-rbac.yaml` | **MODIFY** — add ConfigMap create/update and Secrets read permissions |
| `deploy/helm/tacito-square/values.yaml` | **MODIFY** — document new RBAC entries |
