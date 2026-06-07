# SPEC-FR-M6.5: A2A Agent Cards

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M6.5                                |
| Status        | ACCEPTED                                    |
| Milestone     | M6                                          |
| Component     | agent, keeper                               |
| Depends On    | SPEC-FR-M5.1, SPEC-FR-M6.0                  |
| Supersedes    | none                                        |

## Context

Each agent publishes an Agent Card describing its capabilities, per the A2A protocol. Agent Cards enable discovery and capability-based routing within communities. A centralized registry in Keeper maintains the active agent cards, and agents can query it using NATS Request/Reply or HTTP endpoints.

## Specification

### 1. Agent Card JSON Schema
The Agent Card MUST adhere to the official A2A AgentCard JSON schema. In Tacito Square, it will be defined as follows:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "AgentCard",
  "type": "object",
  "properties": {
    "name": { "type": "string" },
    "description": { "type": "string" },
    "url": { "type": "string" },
    "provider": {
      "type": "object",
      "properties": {
        "organization": { "type": "string" },
        "url": { "type": "string" }
      },
      "required": ["organization", "url"]
    },
    "version": { "type": "string" },
    "documentationUrl": { "type": "string" },
    "capabilities": {
      "type": "object",
      "properties": {
        "streaming": { "type": "boolean" },
        "pushNotifications": { "type": "boolean" },
        "stateTransitionHistory": { "type": "boolean" }
      }
    },
    "authentication": {
      "type": "object",
      "properties": {
        "schemes": {
          "type": "array",
          "items": { "type": "string" }
        },
        "credentials": { "type": "string" }
      },
      "required": ["schemes"]
    },
    "defaultInputModes": {
      "type": "array",
      "items": { "type": "string" }
    },
    "defaultOutputModes": {
      "type": "array",
      "items": { "type": "string" }
    },
    "skills": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "id": { "type": "string" },
          "name": { "type": "string" },
          "description": { "type": "string" },
          "tags": {
            "type": "array",
            "items": { "type": "string" }
          },
          "examples": {
            "type": "array",
            "items": { "type": "string" }
          },
          "inputModes": {
            "type": "array",
            "items": { "type": "string" }
          },
          "outputModes": {
            "type": "array",
            "items": { "type": "string" }
          }
        },
        "required": ["id", "name", "description", "tags"]
      }
    }
  },
  "required": ["name", "description", "url", "version", "capabilities", "authentication", "defaultInputModes", "defaultOutputModes", "skills"]
}
```

### 2. Startup, Heartbeats, and Registry Update
- **Startup:** On startup, the Agent generates its `AgentCard` from its configuration (name, description, brain/model version, loaded skills, etc.).
- **NATS Heartbeats:** Every $N$ seconds (default: 10s, configurable), the agent publishes its complete `AgentCard` payload to NATS on the subject:
  `ts.community.{community_id}.agent.{agent_id}.heartbeat`
- **Registry Update:** The Keeper subscribes to the heartbeat topic wildcard `ts.community.*.agent.*.heartbeat`. Upon receiving a heartbeat:
  - It extracts the `AgentCard` payload and validates it.
  - It saves/updates the card in the database by upserting into the `agent_registrations` table (updating the `card` and setting `last_seen_at = NOW()`).
  - It updates the agent's status to `running` or `active` in the `agents` table.
  - It invalidates/updates the Keeper's Redis cache for that community's active cards.
- **OTel context:** The NATS heartbeat message must carry and propagate the active OpenTelemetry context using `X-Tacito-Tenant` and other standard headers.

### 3. Registry Pruning
- The Keeper maintains a background scheduler/timer task to check for dead agents.
- If no heartbeat is received from an agent within $M$ seconds (default: 30s, configurable) as calculated from `last_seen_at` in the `agent_registrations` table, the Keeper:
  - Updates the agent's status in PostgreSQL `agents` table to `offline`.
  - Deletes the registration row from `agent_registrations`.
  - Clears/removes the agent's card from the Redis active cache.
  - Publishes a NATS status change event to `ts.community.{community_id}.agent.{agent_id}.status` with status `offline` so other agents can invalidate their client-side cache.

### 4. Agent Discovery Query (NATS Request/Reply)
- Spoke/Hub agents can retrieve the active agent cards of their community population asynchronously.
- The agent sends a request on the subject:
  `ts.community.{community_id}.registry.request`
- The Keeper listens to this subject, queries the active cards of the community (from its Redis cache or PostgreSQL), and replies with a JSON list of active `AgentCard` structures.
- **Client-side Cache:** Agents cache the retrieved community cards locally in memory. They listen to status changes and heartbeats to update or invalidate their local cache.

### 5. Keeper HTTP Endpoints & Caching
To remain compliant with standard A2A protocol structures, the registry exposes the `.well-known` path elements scoped by community. To facilitate proxy, gateway, and reverse proxy caching, all of these endpoints MUST serve appropriate HTTP Cache-Control headers supporting public caching with a configurable TTL (default: 30 seconds):

- **Specific Agent Card:**
  `GET /api/v1/communities/{community_id}/agents/{agent_id}/.well-known/agent-card.json`
  - Returns the validated `AgentCard` JSON representation of the specific agent.
  - MUST include header: `Cache-Control: public, max-age=30`
  
- **Community Card (Collective Community Capabilities):**
  `GET /api/v1/communities/{community_id}/.well-known/community-card.json`
  - Returns the collective identity of the community, detailing its configuration, topology, and the list of active member agents (and their card summaries).
  - MUST include header: `Cache-Control: public, max-age=30`
  
- **Community Agent Cards List:**
  `GET /api/v1/communities/{community_id}/.well-known/agent-cards.json`
  - Returns the lightweight index of all active agent cards within the community.
  - MUST include header: `Cache-Control: public, max-age=30`

#### Community Card JSON Structure Example:
```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "community_id": "99f67a21-998f-4cb1-8077-bdcf1262d08a",
  "name": "developer-community",
  "description": "A hub-spoke community of specialized coding and design agents.",
  "topology": "hub-spoke",
  "status": "active",
  "agents": [
    {
      "id": "urn:agent:code-reviewer",
      "name": "code-reviewer",
      "version": "2.1.0",
      "description": "Audits Java/Python code.",
      "capabilities": ["code-analysis"]
    }
  ]
}
```

## Acceptance Criteria

1. **DB Schema:** A separate `agent_registrations` table maps agent IDs and community IDs to their active `card` JSONB data and `last_seen_at` timestamps.
2. **Agent Heartbeat:** On startup and every 10s, the agent publishes its card to `ts.community.{community_id}.agent.{agent_id}.heartbeat` with correct OpenTelemetry headers.
3. **Keeper Registry Ingestion:** Keeper receives the NATS heartbeats, validates the payload against the AgentCard schema, upserts the `agent_registrations` table, updates the `agents` table status to `running`, and caches the card in Redis.
4. **Keeper Pruning:** If an agent's heartbeat is missing for 30s, Keeper deletes the `agent_registrations` record, transitions the status in `agents` to `offline`, and clears its cache.
5. **NATS Discovery:** Agents can execute a NATS Request on `ts.community.{community_id}.registry.request` and receive the community's active agent cards under 10ms.
6. **Local Caching:** Agents cache the retrieved cards in memory and invalidate them reactively when receiving agent status updates from NATS.
7. **HTTP Discovery:**
   - Keeper serves specific cards at `GET /api/v1/communities/{community_id}/agents/{agent_id}/.well-known/agent-card.json`.
   - Keeper serves collective community capabilities at `GET /api/v1/communities/{community_id}/.well-known/community-card.json`.
   - Keeper serves community indexes at `GET /api/v1/communities/{community_id}/.well-known/agent-cards.json`.
   - All three endpoints MUST return the header `Cache-Control: public, max-age=30` (or the configured duration).

## Test Plan

### Automated Tests
1. **Unit Tests:**
   - Verify the parsing and validation of `AgentCard` and `CommunityCard` JSON structures in Go.
   - Verify the local in-memory client-side cache behavior (caching, invalidation, refresh).
2. **Integration Tests (NATS & PostgreSQL):**
   - Spin up a test container with Postgres and NATS.
   - Boot a test keeper service and publish a mock heartbeat. Verify the keeper updates the database status and the `card` JSONB field.
   - Simulate a missing heartbeat and assert that the keeper prunes the agent and updates the status to `offline`.
   - Issue a request/reply message on `ts.community.{community_id}.registry.request` and verify the correct payload response list.
3. **HTTP Endpoint Contract Tests:**
   - Run tests against:
     - `GET /api/v1/communities/{community_id}/agents/{agent_id}/.well-known/agent-card.json`
     - `GET /api/v1/communities/{community_id}/.well-known/community-card.json`
     - `GET /api/v1/communities/{community_id}/.well-known/agent-cards.json`

## Files Affected

- [00001_init.sql](file:///Users/R.Pasquini/Projects/side/tacito-square/deploy/postgres/migrations/00001_init.sql)
- [agent.go (domain model)](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/domain/model/agent.go)
- [community.go (domain model)](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/domain/model/community.go)
- [agent_repository.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/outbound/postgres/agent_repository.go)
- [agent_handlers.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/inbound/http/agent_handlers.go)
- [community_handlers.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/inbound/http/community_handlers.go)
- [bootstrap.go (keeper)](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/bootstrap.go)
- [main.go (agent)](file:///Users/R.Pasquini/Projects/side/tacito-square/cmd/agent/main.go)
- [main.go (keeper)](file:///Users/R.Pasquini/Projects/side/tacito-square/cmd/keeper/main.go)

