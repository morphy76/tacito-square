# E2E Test Strategy — Tacito Square System Lifecycle

> **Scope:** General-purpose end-to-end test strategy covering the full system lifecycle,
> from building images through deploying the cluster to verifying agent conversations.
> This document is the persistent knowledge base for every tester agent run.

---

## 1. Environment Prerequisites

### 1.1 Cluster & Namespace

```bash
# Verify correct context
kubectl config use-context rancher-desktop
kubectl config current-context   # must print: rancher-desktop

# Confirm tacito namespace exists and pods are present
kubectl get pods -n tacito

# Expected running pods after a full deploy:
#   keeper-*        (1 or more replicas)
#   operator-*      (1 replica)
#   agent pods      (0..N — deployed on-demand by the operator)

# Quick readiness gate
kubectl wait --for=condition=ready pod --all -n tacito --timeout=120s
```

### 1.2 Required Tooling on the Host

| Tool | Minimum Version | Purpose |
|------|----------------|---------|
| `kubectl` | any recent | Cluster inspection and log streaming |
| `helm` | v3.x | Chart install / uninstall |
| `docker` | any recent | Image builds |
| `make` | GNU make | Orchestrates all build and deploy steps |
| `curl` | any | REST API interaction |
| `jq` | any | JSON parsing of API responses |
| `uuidgen` | any (macOS built-in) | Generate thread UUIDs |

### 1.3 Workspace Root

All `make` commands must be run from the repository root:

```bash
cd /Users/R.Pasquini/Projects/side/tacito-square
```

---

## 2. Building Images with the Makefile

> Build images only when the codebase has changed since the last deploy.
> Skip this section if the cluster is already running the current commit.

### 2.1 Verify Current Versions

```bash
cat VERSION.keeper   # e.g. 0.5.0
cat VERSION.agent    # e.g. 0.5.0
cat VERSION.operator # e.g. 0.5.0
cat VERSION.bff      # e.g. 0.5.0
```

### 2.2 Build All Docker Images

```bash
# Builds ALL components: agent, keeper, operator, bff, ui-configurator, ollama
make docker-build
```

Or build individual components when only one changed:

```bash
make docker-build-keeper    # keeper only
make docker-build-agent     # agent only
make docker-build-operator  # operator only
make docker-build-bff       # bff only
```

> **Note:** Images are tagged from the `VERSION.*` files automatically.
> No `REGISTRY` prefix is set by default — images land in the local Docker daemon,
> which Rancher Desktop makes available to the cluster directly.

### 2.3 Verify Images Were Built

```bash
docker images | grep tacito-square
# Expected lines like:
#   tacito-square/keeper    0.5.0   ...
#   tacito-square/agent     0.5.0   ...
#   tacito-square/operator  0.5.0   ...
```

---

## 3. Deploying the Cluster with the Makefile

### 3.1 Deploy Infrastructure (First Time Only)

The infrastructure chart installs PostgreSQL, NATS, Redis, and Qdrant.
Only run this if infra is not yet installed or needs upgrading.

```bash
# Download chart dependencies (run once per workstation)
make helm-infra-deps

# Lint the infra chart
make helm-infra-lint

# Preview what would be installed
make helm-infra-template

# Install / upgrade infra release (release name: ts-infra, namespace: tacito)
make helm-infra-install
```

Verify infrastructure is healthy:

```bash
kubectl get pods -n tacito -l app.kubernetes.io/managed-by=Helm
kubectl wait --for=condition=ready pod --all -n tacito --timeout=180s
```

### 3.2 Deploy the Application

```bash
# Preview rendered templates before installing
make helm-template

# Install / upgrade the full application release
# (release name: ts, namespace: tacito, values: tools/helm/dev-values.yaml)
make helm-install
```

This installs keeper, operator, and bff into the `tacito` namespace using
the `--wait` flag — the command blocks until all pods report Ready.

### 3.3 Conditional Rebuild & Redeploy Pattern

When iterating on a specific component:

```bash
# 1. Rebuild only what changed
make docker-build-keeper

# 2. Upgrade the release — Helm picks up the new image tag automatically
make helm-install

# 3. Confirm pods rolled over cleanly
kubectl rollout status deployment/keeper -n tacito
```

### 3.4 Undeploying the Application

```bash
# Remove the application release (keeps the infra release intact)
make helm-uninstall

# To also remove infrastructure (destroys data — use with caution)
make helm-infra-uninstall
```

---

## 4. Health Check Verification

```bash
# Port-forward keeper (run in a dedicated terminal — keep alive for the whole session)
kubectl port-forward -n tacito svc/keeper 8080:8080

# Liveness probe
curl -s http://localhost:8080/healthz | jq .
# Expected: HTTP 200

# Readiness probe (checks postgres + nats + redis + cache-redis in parallel)
curl -s http://localhost:8080/readyz | jq .
# Expected: HTTP 200 with all dependencies showing "ok"

# OpenAPI spec
curl -s http://localhost:8080/openapi.json | jq '.info'
```

---

## 5. Tenant Identity

All Keeper API calls are multitenant. Two headers are **mandatory** on every request.

| Header | Test Value |
|--------|-----------|
| `X-Tenant-ID` | `local` |
| `X-Subscription-ID` | `test` |

Define a shell convenience function at the start of every test session:

```bash
KEEPER=http://localhost:8080

function kcurl() {
  curl -s \
    -H "X-Tenant-ID: local" \
    -H "X-Subscription-ID: test" \
    -H "Content-Type: application/json" \
    "$@"
}
```

---

## 6. Supporting Entity Setup

Create these prerequisite resources once per test session and capture the returned IDs.

### 6.1 Create an LLM Binding

```bash
LLM_BINDING_ID=$(kcurl -i -X POST $KEEPER/api/v1/llm-bindings \
  -d '{
    "name": "test-openai-binding",
    "description": "Test binding for E2E verification",
    "provider": "openai",
    "api_base_url": "https://api.openai.com/v1",
    "api_key_secret_ref": "openai-api-key",
    "default_model": "gpt-4o-mini",
    "default_temperature": 0.7,
    "default_max_tokens": 2048,
    "timeout_seconds": 60
  }' \
  | grep -i location | sed 's|.*llm-bindings/||' | tr -d '\r\n')

echo "LLM Binding ID: $LLM_BINDING_ID"

# Verify — api_key_secret_ref must be scrubbed (empty) in the GET response
kcurl $KEEPER/api/v1/llm-bindings/$LLM_BINDING_ID | jq '.api_key_secret_ref == ""'
# Expected: true
```

> **Note:** `api_key_secret_ref` is a Kubernetes Secret reference name, not the raw API key.
> Any non-empty string is accepted by Keeper. The agent pod will fail to call the LLM
> only if the secret does not exist in the cluster — Keeper CRUD succeeds regardless.

**Verify bad-provider rejection (negative case):**

```bash
kcurl -X POST $KEEPER/api/v1/llm-bindings \
  -d '{"name":"x","provider":"unknown-llm","api_base_url":"http://x","api_key_secret_ref":"x","default_model":"x"}' \
  | jq .
# Expected: HTTP 422 { "error": "invalid provider" }
```

### 6.2 (Optional) Create a Prompt Template

```bash
PROMPT_ID=$(kcurl -i -X POST $KEEPER/api/v1/prompts \
  -d '{
    "name": "test-system-prompt",
    "description": "Baseline system prompt for E2E tests",
    "content": "You are a helpful assistant. Answer concisely."
  }' \
  | grep -i location | sed 's|.*prompts/||' | tr -d '\r\n')

echo "Prompt ID: $PROMPT_ID"
kcurl $KEEPER/api/v1/prompts/$PROMPT_ID | jq '{name, content}'
```

### 6.3 Create an Agent

> Agents must NOT carry a `role` field — role is a community assignment behavior.

```bash
AGENT_PAYLOAD=$(cat <<EOF
{
  "name": "test-agent-standalone",
  "description": "Standalone agent for E2E verification",
  "brain": {
    "llm_binding_id": "$LLM_BINDING_ID",
    "temperature": 0.7,
    "max_tokens": 1024
  },
  "short_term_memory": {
    "key_namespace": "e2e-test",
    "ttl_seconds": 3600
  },
  "long_term_memory": {
    "collection_name": "e2e-test-ltm",
    "vector_dimension": 1536
  }
}
EOF
)

AGENT_ID=$(kcurl -i -X POST $KEEPER/api/v1/agents \
  -d "$AGENT_PAYLOAD" \
  | grep -i location | sed 's|.*agents/||' | tr -d '\r\n')

echo "Agent ID: $AGENT_ID"

# The returned agent must NOT have a top-level "role" field
kcurl $KEEPER/api/v1/agents/$AGENT_ID | jq 'has("role")'
# Expected: false
```

For hub-spoke tests, create two distinct agents:

```bash
HUB_AGENT_ID=$(kcurl -i -X POST $KEEPER/api/v1/agents \
  -d "{
    \"name\": \"test-hub-agent\",
    \"brain\": {\"llm_binding_id\": \"$LLM_BINDING_ID\"},
    \"short_term_memory\": {\"key_namespace\": \"hub-ns\", \"ttl_seconds\": 3600},
    \"long_term_memory\": {\"collection_name\": \"hub-ltm\", \"vector_dimension\": 1536}
  }" \
  | grep -i location | sed 's|.*agents/||' | tr -d '\r\n')

SPOKE_AGENT_ID=$(kcurl -i -X POST $KEEPER/api/v1/agents \
  -d "{
    \"name\": \"test-spoke-agent\",
    \"brain\": {\"llm_binding_id\": \"$LLM_BINDING_ID\"},
    \"short_term_memory\": {\"key_namespace\": \"spoke-ns\", \"ttl_seconds\": 3600},
    \"long_term_memory\": {\"collection_name\": \"spoke-ltm\", \"vector_dimension\": 1536}
  }" \
  | grep -i location | sed 's|.*agents/||' | tr -d '\r\n')

echo "Hub: $HUB_AGENT_ID | Spoke: $SPOKE_AGENT_ID"
```

---

## 7. Community Setup

### 7.1 `single-agent` Topology

```bash
COMMUNITY_ID=$(kcurl -i -X POST $KEEPER/api/v1/communities \
  -d '{
    "name": "e2e-standalone-community",
    "description": "Single-agent community for E2E tests",
    "topology": "single-agent",
    "configuration": {}
  }' \
  | grep -i location | sed 's|.*communities/||' | tr -d '\r\n')

echo "Community ID: $COMMUNITY_ID"
kcurl $KEEPER/api/v1/communities/$COMMUNITY_ID | jq '{id, name, topology, status}'
```

### 7.2 `hub-spoke` Topology

```bash
HS_COMMUNITY_ID=$(kcurl -i -X POST $KEEPER/api/v1/communities \
  -d '{
    "name": "e2e-hubspoke-community",
    "description": "Hub-spoke community for E2E tests",
    "topology": "hub-spoke",
    "configuration": {}
  }' \
  | grep -i location | sed 's|.*communities/||' | tr -d '\r\n')

echo "Hub-Spoke Community ID: $HS_COMMUNITY_ID"
```

---

## 8. Agent Assignment to Community

Role is determined by the assignment — not stored on the agent itself.

### 8.1 Assign to `single-agent` Community

Role is automatically set to `standalone`; any caller-supplied role is ignored.

```bash
kcurl -X POST $KEEPER/api/v1/communities/$COMMUNITY_ID/agents \
  -d "{\"agent_id\": \"$AGENT_ID\"}" | jq .
# Expected: 201 { "agent_id": "...", "role": "standalone", "assigned_at": "..." }
```

### 8.2 Assign Hub and Spoke to `hub-spoke` Community

```bash
# Hub
kcurl -X POST $KEEPER/api/v1/communities/$HS_COMMUNITY_ID/agents \
  -d "{\"agent_id\": \"$HUB_AGENT_ID\", \"role\": \"hub\"}" | jq .
# Expected: 201 with role=hub

# Spoke
kcurl -X POST $KEEPER/api/v1/communities/$HS_COMMUNITY_ID/agents \
  -d "{\"agent_id\": \"$SPOKE_AGENT_ID\", \"role\": \"spoke\"}" | jq .
# Expected: 201 with role=spoke
```

### 8.3 List Assignments

```bash
kcurl $KEEPER/api/v1/communities/$COMMUNITY_ID/agents | jq .
# Expected: array of { agent_id, role, assigned_at, informed_at }

kcurl $KEEPER/api/v1/communities/$HS_COMMUNITY_ID/agents | jq .
# Expected: 2 entries — one hub, one spoke
```

### 8.4 Negative Cases

```bash
# Duplicate hub — must return 409
kcurl -X POST $KEEPER/api/v1/communities/$HS_COMMUNITY_ID/agents \
  -d "{\"agent_id\": \"$AGENT_ID\", \"role\": \"hub\"}" | jq .
# Expected: 409 { "error": "cannot have more than one..." }

# Hub role in single-agent community — must return 409
kcurl -X POST $KEEPER/api/v1/communities/$COMMUNITY_ID/agents \
  -d "{\"agent_id\": \"$AGENT_ID\", \"role\": \"hub\"}" | jq .
# Expected: 409 { "error": "invalid role..." }
```

---

## 9. Community Deployment

Deploying a community triggers the Operator reconciliation loop, which creates agent pods.

```bash
# Trigger deployment
kcurl -X POST $KEEPER/api/v1/communities/$COMMUNITY_ID/deploy | jq .

# Poll status until it reflects the deployed state
kcurl $KEEPER/api/v1/communities/$COMMUNITY_ID/status | jq .

# Verify the operator created the pod(s)
kubectl get pods -n tacito

# Confirm the pod carries the correct role via environment variable
kubectl get pod -n tacito -l tacito.ai/community-id=$COMMUNITY_ID \
  -o jsonpath='{.items[0].spec.containers[0].env}' \
  | jq '.[] | select(.name=="TS_AGENT_ROLE")'
# Expected for standalone: { "name": "TS_AGENT_ROLE", "value": "standalone" }
# Expected for hub:        { "name": "TS_AGENT_ROLE", "value": "hub" }
```

---

## 10. SSE Stream — Observing Community Events

The Keeper SSE endpoint streams all tenant-scoped domain events in real time.
**Open this before sending any conversation events** to capture the full trace.

### 10.1 Open the Stream

```bash
# Run in a dedicated terminal — leave it open for the entire test session
curl -N \
  -H "X-Tenant-ID: local" \
  -H "X-Subscription-ID: test" \
  http://localhost:8080/api/v1/events/stream
```

**SSE message format:**

```
id: <event-uuid>
event: <type>    ← last meaningful segment of the schema URN (see table below)
data: {"event_id":"...","schema_ref":"urn:tacito:...","tenant_id":"local/test","payload":{...}}

```

### 10.2 Event Reference Table

| SSE `event:` value | Full Schema URN | When emitted |
|--------------------|----------------|--------------|
| `start-thread` | `urn:tacito:schema:conversational:start-thread:v1` | Thread opened by the client |
| `add-user-message` | `urn:tacito:schema:conversational:add-user-message:v1` | User turn submitted |
| `agent-response` | `urn:tacito:schema:conversational:agent-response:v1` | Agent final answer (`finished=true`) |
| `agent-reasoning` | `urn:tacito:schema:conversational:agent-reasoning:v1` | Hub reasoning/coordination step |
| `agent-delegation` | `urn:tacito:schema:conversational:agent-delegation:v1` | Hub delegates a task to a spoke |
| `agent-spoke-response` | `urn:tacito:schema:conversational:agent-spoke-response:v1` | Spoke response back to hub |
| `agent-tool-evaluation` | `urn:tacito:schema:conversational:agent-tool-evaluation:v1` | Tool call result |
| `end-thread` | `urn:tacito:schema:conversational:end-thread:v1` | Thread closed by the client |

> `agent-heartbeat` (`urn:tacito:schema:infrastructure:agent-heartbeat:v1`) is blacklisted
> server-side and will never appear in the SSE stream.

### 10.3 Filter a Specific Event Type

```bash
# Terminal 2 — stream only agent-response events
curl -N \
  -H "X-Tenant-ID: local" -H "X-Subscription-ID: test" \
  http://localhost:8080/api/v1/events/stream \
  | grep --line-buffered "^event: agent-response" -A 3
```

---

## 11. Engaging an Agent in a Conversation

All conversational interactions go through the **Keeper event API** (`POST /api/v1/events`).
Keeper publishes to NATS JetStream; the agent pod consumes from its community-scoped subject.

### 11.1 Start a Thread

```bash
THREAD_ID=$(uuidgen | tr '[:upper:]' '[:lower:]')
echo "Thread ID: $THREAD_ID"

kcurl -X POST $KEEPER/api/v1/events \
  -d "{
    \"schema_ref\": \"urn:tacito:schema:conversational:start-thread:v1\",
    \"payload\": {
      \"thread_id\": \"$THREAD_ID\",
      \"community_id\": \"$COMMUNITY_ID\",
      \"metadata\": {\"source\": \"e2e-test\"}
    }
  }" | jq .

# Expected: HTTP 202 Accepted
# Location header: /api/v1/communities/<comm-id>/threads/<thread-id>
# SSE stream: event: start-thread
```

### 11.2 Send a User Message

```bash
kcurl -X POST $KEEPER/api/v1/events \
  -d "{
    \"schema_ref\": \"urn:tacito:schema:conversational:add-user-message:v1\",
    \"payload\": {
      \"thread_id\": \"$THREAD_ID\",
      \"community_id\": \"$COMMUNITY_ID\",
      \"message\": \"Hello, what is the capital of France?\"
    }
  }" | jq .

# Expected: HTTP 202 Accepted
# SSE stream: event: add-user-message  ← fires immediately
# SSE stream: event: agent-response    ← fires when the agent completes the turn
```

> The `agent-response` event with `"finished": true` in the payload signals that
> the agent has completed its turn. For hub-spoke, expect `agent-reasoning`,
> `agent-delegation`, and `agent-spoke-response` events before the final answer.

### 11.3 Multi-turn Follow-up

```bash
kcurl -X POST $KEEPER/api/v1/events \
  -d "{
    \"schema_ref\": \"urn:tacito:schema:conversational:add-user-message:v1\",
    \"payload\": {
      \"thread_id\": \"$THREAD_ID\",
      \"community_id\": \"$COMMUNITY_ID\",
      \"message\": \"Now what is the population of that city?\"
    }
  }" | jq .
```

### 11.4 End the Thread

```bash
kcurl -X POST $KEEPER/api/v1/events \
  -d "{
    \"schema_ref\": \"urn:tacito:schema:conversational:end-thread:v1\",
    \"payload\": {
      \"thread_id\": \"$THREAD_ID\",
      \"community_id\": \"$COMMUNITY_ID\",
      \"reason\": \"e2e-test-complete\"
    }
  }" | jq .

# Expected: HTTP 202 Accepted
# SSE stream: event: end-thread
```

---

## 12. Hub-Spoke Orchestration Verification

For hub-spoke communities, the expected SSE event sequence for a single user turn is:

```
start-thread
  └─> add-user-message
        └─> agent-reasoning       (hub decides how to respond or delegate)
              └─> agent-delegation      (hub assigns task to a spoke)
                    └─> agent-spoke-response   (spoke returns answer to hub)
                          └─> agent-response (finished=true)  (hub delivers final answer)
```

### 12.1 NATS Subject Patterns

| Role | JetStream Subject |
|------|------------------|
| Hub inbound | `ts.community.<comm-id>.agent.hub` |
| Hub spoke-responses | `ts.community.<comm-id>.agent.*.thread.*.response` |
| Spoke inbound | `ts.community.<comm-id>.agent.<agent-name>` |
| Broadcast (all spokes) | `ts.community.<comm-id>.agent.all` |

### 12.2 Inspect NATS via Debug Pod

```bash
# List NATS streams (TACITO_EVENTS, TACITO_DLQ)
kubectl run nats-debug --rm -it --image=natsio/nats-box -n tacito \
  -- nats stream ls --server nats://nats:4222

# Inspect the main event stream
kubectl run nats-debug --rm -it --image=natsio/nats-box -n tacito \
  -- nats stream info TACITO_EVENTS --server nats://nats:4222

# List all durable consumers (one per agent pod)
kubectl run nats-debug --rm -it --image=natsio/nats-box -n tacito \
  -- nats consumer ls TACITO_EVENTS --server nats://nats:4222
```

---

## 13. Well-Known Card Discovery

```bash
# Community card
kcurl $KEEPER/api/v1/communities/$COMMUNITY_ID/.well-known/community-card.json | jq .

# All agent cards for a community
kcurl $KEEPER/api/v1/communities/$COMMUNITY_ID/.well-known/agent-cards.json | jq .

# Individual agent card
kcurl $KEEPER/api/v1/communities/$COMMUNITY_ID/agents/$AGENT_ID/.well-known/agent-card.json | jq .
```

---

## 14. Monitoring & Log Inspection

### 14.1 Keeper Structured Logs

```bash
kubectl logs -n tacito -l app=keeper --tail=100 -f \
  | jq -R 'fromjson? | {level, message, trace_id, community_id, agent_id}'
```

### 14.2 Agent Pod Logs

```bash
# List running agent pods
kubectl get pods -n tacito --field-selector=status.phase=Running -o name | grep agent

# Stream logs for a specific pod
kubectl logs -n tacito <agent-pod-name> --tail=100 -f \
  | jq -R 'fromjson? | {level, message, thread_id, schema_ref, trace_id}'
```

### 14.3 Prometheus Metrics Spot Check

```bash
curl -s http://localhost:8080/metrics | grep 'http_requests_total'
curl -s http://localhost:8080/metrics | grep 'http_request_duration'
curl -s http://localhost:8080/metrics | grep 'go_goroutines'
```

### 14.4 Pod Resource Usage

```bash
kubectl top pods -n tacito
kubectl describe pod -n tacito <pod-name>   # inspect events and conditions
```

---

## 15. Cleanup

```bash
# 1. Undeploy the community (teardown agent pods via operator)
kcurl -X POST $KEEPER/api/v1/communities/$COMMUNITY_ID/undeploy | jq .

# 2. Unassign agent
kcurl -X DELETE $KEEPER/api/v1/communities/$COMMUNITY_ID/agents/$AGENT_ID
# Expected: 204 No Content

# 3. Delete community
kcurl -X DELETE $KEEPER/api/v1/communities/$COMMUNITY_ID
# Expected: 204 No Content

# 4. Delete agent(s)
kcurl -X DELETE $KEEPER/api/v1/agents/$AGENT_ID
kcurl -X DELETE $KEEPER/api/v1/agents/$HUB_AGENT_ID
kcurl -X DELETE $KEEPER/api/v1/agents/$SPOKE_AGENT_ID
# Expected: 204 No Content each

# 5. Delete LLM Binding
kcurl -X DELETE $KEEPER/api/v1/llm-bindings/$LLM_BINDING_ID
# Expected: 204 No Content

# 6. (Optional) Undeploy the application Helm release
make helm-uninstall

# 7. (Optional) Undeploy infrastructure (DESTRUCTIVE — deletes all data)
make helm-infra-uninstall
```

---

## 16. Quick Reference — Makefile Targets

```bash
make help   # print all targets with descriptions
```

| Target | Description |
|--------|-------------|
| `make docker-build` | Build all Docker images |
| `make docker-build-<component>` | Build a single image (keeper, agent, operator, bff) |
| `make helm-infra-deps` | Download infra chart dependencies |
| `make helm-infra-install` | Install/upgrade infrastructure release |
| `make helm-infra-uninstall` | Tear down infrastructure release |
| `make helm-install` | Install/upgrade application release (--wait) |
| `make helm-uninstall` | Remove application release |
| `make helm-template` | Dry-run render application templates |
| `make test` | Unit tests with race detector |
| `make test-integration` | Integration tests (requires Docker) |
| `make test-contract` | OpenAPI contract tests |
| `make lint` | Run linter |
| `make ci` | Full CI pipeline (lint + test + build + docker-build) |

---

## 17. Quick Reference — All Keeper API Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/healthz` | Liveness probe |
| GET | `/readyz` | Readiness probe |
| GET | `/metrics` | Prometheus metrics |
| GET | `/openapi.json` | OpenAPI 3.x spec |
| **LLM Bindings** |||
| POST | `/api/v1/llm-bindings` | Create |
| GET | `/api/v1/llm-bindings` | List |
| GET | `/api/v1/llm-bindings/:id` | Get (key scrubbed) |
| PUT | `/api/v1/llm-bindings/:id` | Update |
| DELETE | `/api/v1/llm-bindings/:id` | Delete |
| **Agents** |||
| POST | `/api/v1/agents` | Create (no `role` field) |
| GET | `/api/v1/agents` | List |
| GET | `/api/v1/agents/:id` | Get |
| PUT | `/api/v1/agents/:id` | Update |
| DELETE | `/api/v1/agents/:id` | Delete |
| POST | `/api/v1/agents/:id/deploy` | Deploy standalone agent |
| POST | `/api/v1/agents/:id/undeploy` | Undeploy standalone agent |
| GET | `/api/v1/agents/:id/status` | Agent status |
| **Communities** |||
| POST | `/api/v1/communities` | Create |
| GET | `/api/v1/communities` | List |
| GET | `/api/v1/communities/:id` | Get |
| PUT | `/api/v1/communities/:id` | Update |
| DELETE | `/api/v1/communities/:id` | Delete |
| POST | `/api/v1/communities/:id/deploy` | Deploy community |
| POST | `/api/v1/communities/:id/undeploy` | Undeploy community |
| GET | `/api/v1/communities/:id/status` | Community status |
| **Assignments** |||
| POST | `/api/v1/communities/:id/agents` | Assign agent (with role) |
| GET | `/api/v1/communities/:id/agents` | List assignments |
| DELETE | `/api/v1/communities/:id/agents/:agent_id` | Unassign agent |
| **Prompts** |||
| POST | `/api/v1/prompts` | Create template |
| GET | `/api/v1/prompts` | List templates |
| GET | `/api/v1/prompts/:id` | Get template |
| PUT | `/api/v1/prompts/:id` | Update |
| DELETE | `/api/v1/prompts/:id` | Delete |
| POST | `/api/v1/prompt-collections` | Create collection |
| GET | `/api/v1/prompt-collections/:id/resolve` | Resolve collection |
| **Skills** |||
| POST | `/api/v1/skills` | Create skill |
| GET | `/api/v1/skills` | List |
| POST | `/api/v1/agents/:id/skills/:skill_id` | Attach to agent |
| DELETE | `/api/v1/agents/:id/skills/:skill_id` | Detach from agent |
| GET | `/api/v1/skill-collections/:id/resolve` | Resolve collection |
| **Events** |||
| POST | `/api/v1/events` | Publish domain event |
| GET | `/api/v1/events/stream` | SSE stream (tenant-scoped) |
| **Discovery** |||
| GET | `/api/v1/communities/:id/.well-known/community-card.json` | Community A2A card |
| GET | `/api/v1/communities/:id/.well-known/agent-cards.json` | All agent cards |
| GET | `/api/v1/communities/:id/agents/:agent_id/.well-known/agent-card.json` | Individual agent card |
